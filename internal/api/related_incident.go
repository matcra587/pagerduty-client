package api

import (
	"context"
	"encoding/json"
	"fmt"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// Relationship describes how two incidents are related.
// Metadata varies by type:
//   - machine_learning_inferred: {"grouping_classification": "similar_contents", ...}
//   - service_dependency: {"dependent_services": [...], "supporting_services": [...]}
type Relationship struct {
	Type     string         `json:"type"`
	Metadata map[string]any `json:"metadata"`
}

// RelatedIncident is an active incident related to the queried incident.
type RelatedIncident struct {
	Incident      pagerduty.Incident `json:"incident"`
	Relationships []Relationship     `json:"relationships"`
}

// ListRelatedIncidents returns related active incidents for the given incident.
// The endpoint is not paginated; it returns up to 20 items.
func (c *Client) ListRelatedIncidents(ctx context.Context, incidentID string) ([]RelatedIncident, error) {
	path := fmt.Sprintf("/incidents/%s/related_incidents", incidentID)
	body, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		RelatedIncidents []RelatedIncident `json:"related_incidents"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding related incidents: %w", err)
	}
	return resp.RelatedIncidents, nil
}
