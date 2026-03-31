package api

import (
	"context"
	"encoding/json"
	"fmt"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListIncidentNotes returns all notes for the given incident.
// This endpoint is not paginated; all notes are returned in a single response.
func (c *Client) ListIncidentNotes(ctx context.Context, incidentID string) ([]pagerduty.IncidentNote, error) {
	path := fmt.Sprintf("/incidents/%s/notes", incidentID)
	body, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Notes []pagerduty.IncidentNote `json:"notes"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding notes: %w", err)
	}
	return resp.Notes, nil
}
