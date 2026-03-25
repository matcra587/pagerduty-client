package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListTeamsOpts filters for listing teams.
type ListTeamsOpts struct {
	Query string
}

// ListTeams returns all teams matching opts, auto-paginating.
func (c *Client) ListTeams(ctx context.Context, opts ListTeamsOpts) ([]pagerduty.Team, error) {
	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
	}

	var teams []pagerduty.Team
	err := paginate(ctx, c, paginateRequest{
		path:   "/teams",
		params: params,
		key:    "teams",
	}, func(page []pagerduty.Team) {
		teams = append(teams, page...)
	})
	if err != nil {
		return nil, err
	}
	return teams, nil
}

// GetTeam retrieves a single team by ID.
func (c *Client) GetTeam(ctx context.Context, id string) (*pagerduty.Team, error) {
	body, err := c.get(ctx, "/teams/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Team pagerduty.Team `json:"team"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding team: %w", err)
	}
	return &resp.Team, nil
}
