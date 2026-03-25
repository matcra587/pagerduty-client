package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// ListUsersOpts filters for listing users.
type ListUsersOpts struct {
	TeamIDs []string
	Query   string
}

// ListUsers returns all users matching opts, auto-paginating.
func (c *Client) ListUsers(ctx context.Context, opts ListUsersOpts) ([]pagerduty.User, error) {
	params := url.Values{}
	if opts.Query != "" {
		params.Set("query", opts.Query)
	}
	for _, id := range opts.TeamIDs {
		params.Add("team_ids[]", id)
	}

	var users []pagerduty.User
	err := paginate(ctx, c, paginateRequest{
		path:   "/users",
		params: params,
		key:    "users",
	}, func(page []pagerduty.User) {
		users = append(users, page...)
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetUser retrieves a single user by ID.
func (c *Client) GetUser(ctx context.Context, id string) (*pagerduty.User, error) {
	body, err := c.get(ctx, "/users/"+id, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		User pagerduty.User `json:"user"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding user: %w", err)
	}
	return &resp.User, nil
}

// GetCurrentUser retrieves the user associated with the current API token.
func (c *Client) GetCurrentUser(ctx context.Context) (*pagerduty.User, error) {
	body, err := c.get(ctx, "/users/me", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		User pagerduty.User `json:"user"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding user: %w", err)
	}
	return &resp.User, nil
}

// ListContactMethods returns all contact methods for the given user, auto-paginating.
func (c *Client) ListContactMethods(ctx context.Context, userID string) ([]pagerduty.ContactMethod, error) {
	var methods []pagerduty.ContactMethod
	err := paginate(ctx, c, paginateRequest{
		path: "/users/" + userID + "/contact_methods",
		key:  "contact_methods",
	}, func(page []pagerduty.ContactMethod) {
		methods = append(methods, page...)
	})
	if err != nil {
		return nil, err
	}
	return methods, nil
}
