package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListMaintenanceWindowsOpts configures the ListMaintenanceWindows request.
type ListMaintenanceWindowsOpts struct {
	Query      string
	TeamIDs    []string
	ServiceIDs []string
	Filter     string
}

// ListMaintenanceWindows returns all maintenance windows matching opts,
// auto-paginating through all pages.
func (c *Client) ListMaintenanceWindows(ctx context.Context, opts ListMaintenanceWindowsOpts) ([]pagerduty.MaintenanceWindow, error) {
	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
	}
	if opts.Filter != "" {
		params.Set("filter", opts.Filter)
	}
	for _, id := range opts.TeamIDs {
		params.Add("team_ids[]", id)
	}
	for _, id := range opts.ServiceIDs {
		params.Add("service_ids[]", id)
	}

	var windows []pagerduty.MaintenanceWindow
	err := paginate(ctx, c, paginateRequest{
		path:   "/maintenance_windows",
		params: params,
		key:    "maintenance_windows",
	}, func(page []pagerduty.MaintenanceWindow) {
		windows = append(windows, page...)
	})
	if err != nil {
		return nil, err
	}

	return windows, nil
}

// GetMaintenanceWindow returns a single maintenance window by ID.
func (c *Client) GetMaintenanceWindow(ctx context.Context, id string) (*pagerduty.MaintenanceWindow, error) {
	body, err := c.get(ctx, "/maintenance_windows/"+id, nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		MaintenanceWindow pagerduty.MaintenanceWindow `json:"maintenance_window"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("decoding maintenance window: %w", err)
	}

	return &envelope.MaintenanceWindow, nil
}
