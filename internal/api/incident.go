package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListIncidentsOpts filters for listing incidents.
type ListIncidentsOpts struct {
	Statuses   []string
	TeamIDs    []string
	UserIDs    []string
	ServiceIDs []string
	Urgencies  []string
	SortBy     string
	DateRange  string
	Since      string
	Until      string
}

// ListIncidents returns all incidents matching opts, auto-paginating.
func (c *Client) ListIncidents(ctx context.Context, opts ListIncidentsOpts) ([]pagerduty.Incident, error) {
	params := incidentListParams(opts)
	var incidents []pagerduty.Incident
	err := paginate(ctx, c, paginateRequest{
		path:   "/incidents",
		params: params,
		key:    "incidents",
	}, func(page []pagerduty.Incident) {
		incidents = append(incidents, page...)
	})
	if err != nil {
		return nil, err
	}
	return incidents, nil
}

func incidentListParams(opts ListIncidentsOpts) url.Values {
	v := url.Values{}
	for _, s := range opts.Statuses {
		v.Add("statuses[]", s)
	}
	for _, u := range opts.Urgencies {
		v.Add("urgencies[]", u)
	}
	for _, id := range opts.TeamIDs {
		v.Add("team_ids[]", id)
	}
	for _, id := range opts.UserIDs {
		v.Add("user_ids[]", id)
	}
	for _, id := range opts.ServiceIDs {
		v.Add("service_ids[]", id)
	}
	if opts.SortBy != "" {
		v.Set("sort_by", opts.SortBy)
	}
	// date_range and since/until are mutually exclusive: PD ignores
	// since/until when date_range=all is set.
	if opts.DateRange != "" {
		v.Set("date_range", opts.DateRange)
	} else {
		if opts.Since != "" {
			v.Set("since", opts.Since)
		}
		if opts.Until != "" {
			v.Set("until", opts.Until)
		}
	}
	return v
}

// GetIncident retrieves a single incident by ID.
func (c *Client) GetIncident(ctx context.Context, id string) (*pagerduty.Incident, error) {
	body, err := c.get(ctx, "/incidents/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Incident pagerduty.Incident `json:"incident"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding incident: %w", err)
	}
	return &resp.Incident, nil
}

type incidentStatusUpdate struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// AckIncident acknowledges a single incident.
func (c *Client) AckIncident(ctx context.Context, id, from string) error {
	payload := map[string][]incidentStatusUpdate{
		"incidents": {
			{ID: id, Type: "incident_reference", Status: "acknowledged"},
		},
	}
	_, err := c.putFrom(ctx, "/incidents", payload, from)
	return err
}

// ResolveIncident resolves a single incident.
func (c *Client) ResolveIncident(ctx context.Context, id, from string) error {
	payload := map[string][]incidentStatusUpdate{
		"incidents": {
			{ID: id, Type: "incident_reference", Status: "resolved"},
		},
	}
	_, err := c.putFrom(ctx, "/incidents", payload, from)
	return err
}

// SnoozeIncident snoozes an incident for the given duration.
func (c *Client) SnoozeIncident(ctx context.Context, id, from string, duration time.Duration) error {
	if duration <= 0 {
		return errors.New("snooze duration must be positive")
	}
	payload := map[string]int{
		"duration": int(duration.Seconds()),
	}
	_, err := c.postFrom(ctx, "/incidents/"+id+"/snooze", payload, from)
	return err
}

type assigneeRef struct {
	Assignee struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"assignee"`
}

// ReassignIncident reassigns an incident to the given user IDs.
func (c *Client) ReassignIncident(ctx context.Context, id, from string, userIDs []string) error {
	if len(userIDs) == 0 {
		return errors.New("at least one user ID is required")
	}
	assignments := make([]assigneeRef, len(userIDs))
	for i, uid := range userIDs {
		assignments[i].Assignee.ID = uid
		assignments[i].Assignee.Type = "user_reference"
	}

	type incidentReassign struct {
		ID          string        `json:"id"`
		Type        string        `json:"type"`
		Assignments []assigneeRef `json:"assignments"`
	}

	payload := map[string][]incidentReassign{
		"incidents": {
			{ID: id, Type: "incident_reference", Assignments: assignments},
		},
	}
	_, err := c.putFrom(ctx, "/incidents", payload, from)
	return err
}

type sourceIncidentRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// MergeIncidents merges sourceIDs into targetID.
func (c *Client) MergeIncidents(ctx context.Context, targetID, from string, sourceIDs []string) error {
	if len(sourceIDs) == 0 {
		return errors.New("at least one source incident ID is required")
	}
	sources := make([]sourceIncidentRef, len(sourceIDs))
	for i, sid := range sourceIDs {
		sources[i] = sourceIncidentRef{ID: sid, Type: "incident_reference"}
	}

	payload := map[string][]sourceIncidentRef{
		"source_incidents": sources,
	}
	_, err := c.putFrom(ctx, "/incidents/"+targetID+"/merge", payload, from)
	return err
}

type incidentEscalationInfo struct {
	EscalationPolicy struct {
		ID string `json:"id"`
	} `json:"escalation_policy"`
	EscalationLevel uint `json:"escalation_level"`
}

// EscalateIncident reassigns an incident to the next escalation policy level.
// PagerDuty has no direct escalate endpoint, so this method:
//  1. Fetches the incident to read its current escalation_level and EP ID
//  2. Fetches the escalation policy to get all rules
//  3. Finds the next rule (current_level + 1)
//  4. Extracts target user/schedule IDs from the next rule
//  5. Reassigns the incident to those targets
func (c *Client) EscalateIncident(ctx context.Context, id, from string) error {
	if from == "" {
		return errors.New("from email is required for write operations")
	}
	body, err := c.get(ctx, "/incidents/"+id, nil)
	if err != nil {
		return fmt.Errorf("fetching incident for escalation: %w", err)
	}

	var incResp struct {
		Incident incidentEscalationInfo `json:"incident"`
	}
	if err := json.Unmarshal(body, &incResp); err != nil {
		return fmt.Errorf("decoding incident escalation info: %w", err)
	}

	epID := incResp.Incident.EscalationPolicy.ID
	if epID == "" {
		return errors.New("incident has no escalation policy")
	}
	currentLevel := incResp.Incident.EscalationLevel

	ep, err := c.getEscalationPolicy(ctx, epID)
	if err != nil {
		return fmt.Errorf("fetching escalation policy: %w", err)
	}

	// PD returns escalation_level 0 for incidents not yet escalated.
	// Level 0 and level 1 both represent "at the first escalation rule",
	// so we treat them identically and escalate to the next level.
	if currentLevel == 0 {
		currentLevel = 1
	}

	if int(currentLevel) >= len(ep.EscalationRules) {
		return fmt.Errorf("already at highest escalation level (%d of %d)", currentLevel, len(ep.EscalationRules))
	}

	nextRule := ep.EscalationRules[currentLevel]

	var targetIDs []string
	for _, target := range nextRule.Targets {
		targetIDs = append(targetIDs, target.ID)
	}
	if len(targetIDs) == 0 {
		return errors.New("next escalation level has no targets")
	}

	return c.ReassignIncident(ctx, id, from, targetIDs)
}

// getEscalationPolicy fetches an escalation policy by ID. This is a private
// helper for EscalateIncident; the full public method will live in
// escalation_policy.go.
func (c *Client) getEscalationPolicy(ctx context.Context, id string) (*pagerduty.EscalationPolicy, error) {
	body, err := c.get(ctx, "/escalation_policies/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		EscalationPolicy pagerduty.EscalationPolicy `json:"escalation_policy"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding escalation policy: %w", err)
	}
	return &resp.EscalationPolicy, nil
}

// ListPriorities fetches all configured priorities.
func (c *Client) ListPriorities(ctx context.Context) ([]pagerduty.Priority, error) {
	body, err := c.get(ctx, "/priorities", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Priorities []pagerduty.Priority `json:"priorities"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding priorities: %w", err)
	}
	return resp.Priorities, nil
}

// UpdatePriority sets the priority on an incident.
func (c *Client) UpdatePriority(ctx context.Context, id, from, priorityID string) error {
	type priorityRef struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	type incidentUpdate struct {
		ID       string      `json:"id"`
		Type     string      `json:"type"`
		Priority priorityRef `json:"priority"`
	}

	payload := map[string][]incidentUpdate{
		"incidents": {
			{
				ID:       id,
				Type:     "incident_reference",
				Priority: priorityRef{ID: priorityID, Type: "priority_reference"},
			},
		},
	}
	_, err := c.putFrom(ctx, "/incidents", payload, from)
	return err
}

// AddIncidentNote adds a note to an incident.
func (c *Client) AddIncidentNote(ctx context.Context, id, from, content string) error {
	payload := map[string]map[string]string{
		"note": {"content": content},
	}
	_, err := c.postFrom(ctx, "/incidents/"+id+"/notes", payload, from)
	return err
}
