package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListServicesOpts filters for listing services.
type ListServicesOpts struct {
	TeamIDs []string
	Query   string
	SortBy  string
}

// ListServices returns all services matching opts, auto-paginating.
func (c *Client) ListServices(ctx context.Context, opts ListServicesOpts) ([]pagerduty.Service, error) {
	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
	}
	if opts.SortBy != "" {
		params.Set("sort_by", opts.SortBy)
	}
	for _, id := range opts.TeamIDs {
		params.Add("team_ids[]", id)
	}

	var services []pagerduty.Service
	err := paginate(ctx, c, paginateRequest{
		path:   "/services",
		params: params,
		key:    "services",
	}, func(page []pagerduty.Service) {
		services = append(services, page...)
	})
	if err != nil {
		return nil, err
	}
	return services, nil
}

// GetService retrieves a single service by ID.
func (c *Client) GetService(ctx context.Context, id string) (*pagerduty.Service, error) {
	body, err := c.get(ctx, "/services/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Service pagerduty.Service `json:"service"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding service: %w", err)
	}
	return &resp.Service, nil
}
