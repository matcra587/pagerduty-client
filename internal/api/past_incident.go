package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// PastIncident is a historically similar incident returned by the
// PagerDuty past_incidents endpoint.
type PastIncident struct {
	Incident pagerduty.Incident `json:"incident"`
	Score    float64            `json:"score"`
}

// ListPastIncidents returns similar past incidents for the given incident.
// The endpoint is not paginated; limit caps the result count (default 5, max 999).
func (c *Client) ListPastIncidents(ctx context.Context, incidentID string, limit int) ([]PastIncident, error) {
	if limit <= 0 {
		limit = 5
	}
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))

	path := fmt.Sprintf("/incidents/%s/past_incidents", incidentID)
	body, err := c.get(ctx, path, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		PastIncidents []PastIncident `json:"past_incidents"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding past incidents: %w", err)
	}
	return resp.PastIncidents, nil
}
