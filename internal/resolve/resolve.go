// Package resolve provides fuzzy name-to-ID resolution for PagerDuty resources.
package resolve

import (
	"context"
	"fmt"
	"regexp"

	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/sahilm/fuzzy"
)

// idPattern matches PagerDuty resource IDs. Real IDs are 7+
// uppercase alphanumeric characters (e.g. PSVC001, PT4KHLK).
// Shorter all-caps strings like "DB" or "API" fall through to
// fuzzy search, which still resolves them correctly.
var idPattern = regexp.MustCompile(`^[A-Z0-9]{7,}$`)

// Match represents a fuzzy match result.
type Match struct {
	ID   string
	Name string
}

// Resolver resolves user input (ID or partial name) to a PagerDuty resource ID.
type Resolver struct {
	client *api.Client
}

// New returns a Resolver backed by the given API client.
func New(client *api.Client) *Resolver {
	return &Resolver{client: client}
}

// resolve is the core logic shared by all resource methods.
// Returns (id, nil, nil) on single match or ID passthrough.
// Returns ("", matches, nil) on multiple matches.
// Returns ("", nil, error) on zero matches or fetch failure.
func (r *Resolver) resolve(
	ctx context.Context,
	kind, input string,
	fetch func(context.Context) ([]Match, error),
) (string, []Match, error) {
	if idPattern.MatchString(input) {
		return input, nil, nil
	}

	candidates, err := fetch(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("listing %ss: %w", kind, err)
	}

	names := make([]string, len(candidates))
	for i, c := range candidates {
		names[i] = c.Name
	}

	results := fuzzy.Find(input, names)

	// Discard low-quality matches to avoid surprising results.
	filtered := results[:0]
	for _, r := range results {
		if r.Score >= 0 {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		return "", nil, fmt.Errorf("no %s matching %q", kind, input)
	}

	if len(filtered) == 1 {
		return candidates[filtered[0].Index].ID, nil, nil
	}

	matched := make([]Match, len(filtered))
	for i, r := range filtered {
		matched[i] = candidates[r.Index]
	}

	return "", matched, nil
}

// Service resolves a service ID or name.
func (r *Resolver) Service(ctx context.Context, input string) (string, []Match, error) {
	return r.resolve(ctx, "service", input, func(ctx context.Context) ([]Match, error) {
		services, err := r.client.ListServices(ctx, api.ListServicesOpts{})
		if err != nil {
			return nil, err
		}

		matches := make([]Match, len(services))
		for i, s := range services {
			matches[i] = Match{ID: s.ID, Name: s.Name}
		}

		return matches, nil
	})
}

// Team resolves a team ID or name.
func (r *Resolver) Team(ctx context.Context, input string) (string, []Match, error) {
	return r.resolve(ctx, "team", input, func(ctx context.Context) ([]Match, error) {
		teams, err := r.client.ListTeams(ctx, api.ListTeamsOpts{})
		if err != nil {
			return nil, err
		}

		matches := make([]Match, len(teams))
		for i, t := range teams {
			matches[i] = Match{ID: t.ID, Name: t.Name}
		}

		return matches, nil
	})
}

// User resolves a user ID, name or email.
func (r *Resolver) User(ctx context.Context, input string) (string, []Match, error) {
	return r.resolve(ctx, "user", input, func(ctx context.Context) ([]Match, error) {
		users, err := r.client.ListUsers(ctx, api.ListUsersOpts{})
		if err != nil {
			return nil, err
		}

		matches := make([]Match, len(users))
		for i, u := range users {
			matches[i] = Match{ID: u.ID, Name: fmt.Sprintf("%s <%s>", u.Name, u.Email)}
		}

		return matches, nil
	})
}

// Schedule resolves a schedule ID or name.
func (r *Resolver) Schedule(ctx context.Context, input string) (string, []Match, error) {
	return r.resolve(ctx, "schedule", input, func(ctx context.Context) ([]Match, error) {
		schedules, err := r.client.ListSchedules(ctx, api.ListSchedulesOpts{})
		if err != nil {
			return nil, err
		}

		matches := make([]Match, len(schedules))
		for i, s := range schedules {
			matches[i] = Match{ID: s.ID, Name: s.Name}
		}

		return matches, nil
	})
}

// EscalationPolicy resolves an escalation policy ID or name.
func (r *Resolver) EscalationPolicy(ctx context.Context, input string) (string, []Match, error) {
	return r.resolve(ctx, "escalation policy", input, func(ctx context.Context) ([]Match, error) {
		policies, err := r.client.ListEscalationPolicies(ctx, api.ListEscalationPoliciesOpts{})
		if err != nil {
			return nil, err
		}

		matches := make([]Match, len(policies))
		for i, p := range policies {
			matches[i] = Match{ID: p.ID, Name: p.Name}
		}

		return matches, nil
	})
}

// Incident resolves an incident ID or title (open incidents only).
func (r *Resolver) Incident(ctx context.Context, input string) (string, []Match, error) {
	return r.resolve(ctx, "incident", input, func(ctx context.Context) ([]Match, error) {
		incidents, err := r.client.ListIncidents(ctx, api.ListIncidentsOpts{
			Statuses: []string{"triggered", "acknowledged"},
		})
		if err != nil {
			return nil, err
		}

		matches := make([]Match, len(incidents))
		for i, inc := range incidents {
			matches[i] = Match{ID: inc.ID, Name: inc.Title}
		}

		return matches, nil
	})
}

// MaintenanceWindow resolves a maintenance window ID or description.
func (r *Resolver) MaintenanceWindow(ctx context.Context, input string) (string, []Match, error) {
	return r.resolve(ctx, "maintenance window", input, func(ctx context.Context) ([]Match, error) {
		windows, err := r.client.ListMaintenanceWindows(ctx, api.ListMaintenanceWindowsOpts{})
		if err != nil {
			return nil, err
		}

		matches := make([]Match, len(windows))
		for i, w := range windows {
			matches[i] = Match{ID: w.ID, Name: w.Description}
		}

		return matches, nil
	})
}
