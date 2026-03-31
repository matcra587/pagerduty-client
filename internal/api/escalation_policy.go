package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListEscalationPoliciesOpts configures the ListEscalationPolicies request.
type ListEscalationPoliciesOpts struct {
	Query   string
	TeamIDs []string
}

// ListEscalationPolicies returns all escalation policies matching opts,
// auto-paginating through all pages.
func (c *Client) ListEscalationPolicies(ctx context.Context, opts ListEscalationPoliciesOpts) ([]pagerduty.EscalationPolicy, error) {
	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
	}
	for _, id := range opts.TeamIDs {
		params.Add("team_ids[]", id)
	}

	var policies []pagerduty.EscalationPolicy
	err := paginate(ctx, c, paginateRequest{
		path:   "/escalation_policies",
		params: params,
		key:    "escalation_policies",
	}, func(page []pagerduty.EscalationPolicy) {
		policies = append(policies, page...)
	})
	if err != nil {
		return nil, err
	}

	return policies, nil
}

// GetEscalationPolicy returns a single escalation policy by ID.
// Passes include[]=targets to populate target Summary fields.
func (c *Client) GetEscalationPolicy(ctx context.Context, id string) (*pagerduty.EscalationPolicy, error) {
	params := url.Values{}
	params.Add("include[]", "targets")

	body, err := c.get(ctx, "/escalation_policies/"+id, params)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		EscalationPolicy pagerduty.EscalationPolicy `json:"escalation_policy"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decoding escalation policy: %w", err)
	}

	return &envelope.EscalationPolicy, nil
}
