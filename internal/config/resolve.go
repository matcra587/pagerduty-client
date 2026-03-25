package config

import (
	"fmt"
	"strings"
	"unicode"
)

// Team is a lightweight team reference for resolution.
type Team struct {
	ID   string
	Name string
}

// IsTeamID returns true if s looks like a PagerDuty team ID
// (uppercase alphanumeric prefixed with P).
func IsTeamID(s string) bool {
	if len(s) < 2 {
		return false
	}
	if s[0] != 'P' {
		return false
	}
	for _, r := range s[1:] {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ResolveTeam converts a team name or ID into a team ID. If value is
// already an ID it is returned directly. Otherwise listFn is called to
// search by name; exactly one match is required.
func ResolveTeam(value string, listFn func(query string) ([]Team, error)) (string, error) {
	if IsTeamID(value) {
		return value, nil
	}
	teams, err := listFn(value)
	if err != nil {
		return "", err
	}
	switch len(teams) {
	case 1:
		return teams[0].ID, nil
	case 0:
		return "", fmt.Errorf("no team found matching %q", value)
	default:
		names := make([]string, len(teams))
		for i, t := range teams {
			names[i] = fmt.Sprintf("%s (%s)", t.Name, t.ID)
		}
		return "", fmt.Errorf("ambiguous team name %q - matches: %s", value, strings.Join(names, ", "))
	}
}
