package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// ListAbilities returns the account's enabled abilities. This is a
// lightweight, non-paginated endpoint that works with both user tokens
// and account-level API keys, making it suitable for token validation.
func (c *Client) ListAbilities(ctx context.Context) ([]string, error) {
	body, err := c.get(ctx, "/abilities", nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Abilities []string `json:"abilities"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding abilities: %w", err)
	}
	return resp.Abilities, nil
}
