package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Ability represents a PagerDuty account ability.
// The API returns abilities as bare strings (e.g. "service_support_hours").
type Ability struct {
	Name    string `json:"name"`
	Display string `json:"display"`
}

// displayAcronyms maps lowercase words to their preferred casing.
var displayAcronyms = map[string]string{
	"api": "API", "ios": "iOS", "sso": "SSO",
	"ui": "UI", "url": "URL", "id": "ID",
	"v1": "v1", "v2": "v2", "v3": "v3",
}

// toAbilities converts raw ability strings into Ability values with
// human-readable display names.
func toAbilities(names []string) []Ability {
	tc := cases.Title(language.English)
	abilities := make([]Ability, len(names))
	for i, n := range names {
		abilities[i] = Ability{
			Name:    n,
			Display: humanise(tc, n),
		}
	}
	return abilities
}

// humanise converts a snake_case ability name into a title-cased
// display string, applying acronym overrides where appropriate.
func humanise(tc cases.Caser, name string) string {
	words := strings.Split(name, "_")
	for i, w := range words {
		if override, ok := displayAcronyms[w]; ok {
			words[i] = override
		} else {
			words[i] = tc.String(w)
		}
	}
	return strings.Join(words, " ")
}

// ListAbilities returns the account's enabled abilities. This is a
// lightweight, non-paginated endpoint that works with both user tokens
// and account-level API keys, making it suitable for token validation.
func (c *Client) ListAbilities(ctx context.Context) ([]Ability, error) {
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
	return toAbilities(resp.Abilities), nil
}

// TestAbility checks whether the account has the named ability.
// Returns nil if the ability is available (HTTP 204).
// Returns an *APIError with status 402 if the ability is unavailable,
// or 404 if the ability name is unrecognised.
func (c *Client) TestAbility(ctx context.Context, name string) error {
	_, err := c.get(ctx, "/abilities/"+name, nil)
	return err
}
