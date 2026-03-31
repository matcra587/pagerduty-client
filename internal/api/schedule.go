package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListSchedulesOpts configures the ListSchedules request.
type ListSchedulesOpts struct {
	Query string
}

// CreateOverrideOpts configures the CreateOverride request.
// Start and End must be ISO 8601 timestamps.
type CreateOverrideOpts struct {
	UserID string
	Start  string
	End    string
}

// ListSchedules returns all on-call schedules, auto-paginating through all pages.
func (c *Client) ListSchedules(ctx context.Context, opts ListSchedulesOpts) ([]pagerduty.Schedule, error) {
	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
	}

	var schedules []pagerduty.Schedule
	err := paginate[pagerduty.Schedule](ctx, c, paginateRequest{
		path:   "/schedules",
		params: params,
		key:    "schedules",
	}, func(page []pagerduty.Schedule) {
		schedules = append(schedules, page...)
	})
	if err != nil {
		return nil, err
	}

	return schedules, nil
}

// GetSchedule returns detailed information about a single schedule by ID.
func (c *Client) GetSchedule(ctx context.Context, id string) (*pagerduty.Schedule, error) {
	body, err := c.get(ctx, "/schedules/"+id, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Schedule pagerduty.Schedule `json:"schedule"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decoding schedule: %w", err)
	}

	return &envelope.Schedule, nil
}

// ListOverrides returns all overrides for a schedule within the given time window.
// since and until must be ISO 8601 formatted timestamps.
// This endpoint is not paginated.
func (c *Client) ListOverrides(ctx context.Context, scheduleID, since, until string) ([]pagerduty.Override, error) {
	params := url.Values{}
	params.Set("since", since)
	params.Set("until", until)

	body, err := c.get(ctx, "/schedules/"+scheduleID+"/overrides", params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Overrides []pagerduty.Override `json:"overrides"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding overrides: %w", err)
	}
	return resp.Overrides, nil
}

type overridesPayload struct {
	Overrides []overrideBody `json:"overrides"`
}

type overrideBody struct {
	Start string              `json:"start"`
	End   string              `json:"end"`
	User  pagerduty.APIObject `json:"user"`
}

type overrideResult struct {
	Status   int                `json:"status"`
	Override pagerduty.Override `json:"override"`
}

// CreateOverride creates an override for a schedule, placing the specified user
// on call for the duration defined by opts.Start and opts.End.
// Uses the bulk overrides endpoint (POST /schedules/{id}/overrides).
// The from parameter is the email of the acting user (required by the PD API).
func (c *Client) CreateOverride(ctx context.Context, scheduleID, from string, opts CreateOverrideOpts) error {
	payload := overridesPayload{
		Overrides: []overrideBody{
			{
				Start: opts.Start,
				End:   opts.End,
				User: pagerduty.APIObject{
					ID:   opts.UserID,
					Type: "user_reference",
				},
			},
		},
	}

	body, err := c.postFrom(ctx, "/schedules/"+scheduleID+"/overrides", payload, from)
	if err != nil {
		return err
	}

	var results []overrideResult
	if err := json.Unmarshal(body, &results); err != nil {
		return fmt.Errorf("decoding override response: %w", err)
	}

	if len(results) == 0 {
		return errors.New("empty override response")
	}

	if results[0].Status != http.StatusCreated {
		return fmt.Errorf("override creation failed with status %d", results[0].Status)
	}

	return nil
}
