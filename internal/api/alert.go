package api

import (
	"context"
	"encoding/json"
	"errors"
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

// ResolveAlerts resolves one or more alerts within an incident.
func (c *Client) ResolveAlerts(ctx context.Context, incidentID, from string, alertIDs []string) error {
	if len(alertIDs) == 0 {
		return errors.New("at least one alert ID is required")
	}

	type alertUpdate struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}

	alerts := make([]alertUpdate, len(alertIDs))
	for i, id := range alertIDs {
		alerts[i] = alertUpdate{ID: id, Type: "alert", Status: "resolved"}
	}

	payload := map[string][]alertUpdate{
		"alerts": alerts,
	}

	path := fmt.Sprintf("/incidents/%s/alerts", incidentID)
	_, err := c.putFrom(ctx, path, payload, from)
	return err
}
