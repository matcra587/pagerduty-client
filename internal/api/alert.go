package api

import (
	"context"
	"encoding/json"
	"fmt"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListIncidentAlerts returns all alerts for the given incident, auto-paginating.
func (c *Client) ListIncidentAlerts(ctx context.Context, incidentID string) ([]pagerduty.IncidentAlert, error) {
	path := fmt.Sprintf("/incidents/%s/alerts", incidentID)
	var alerts []pagerduty.IncidentAlert
	err := paginate(ctx, c, paginateRequest{
		path: path,
		key:  "alerts",
	}, func(page []pagerduty.IncidentAlert) {
		alerts = append(alerts, page...)
	})
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// GetIncidentAlert retrieves a single alert by incident and alert ID.
func (c *Client) GetIncidentAlert(ctx context.Context, incidentID, alertID string) (*pagerduty.IncidentAlert, error) {
	path := fmt.Sprintf("/incidents/%s/alerts/%s", incidentID, alertID)
	body, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Alert pagerduty.IncidentAlert `json:"alert"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding alert: %w", err)
	}
	return &resp.Alert, nil
}
