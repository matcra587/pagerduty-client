package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gechr/clib/complete"
	"github.com/matcra587/pagerduty-client/internal/api"
	"github.com/matcra587/pagerduty-client/internal/config"
)

// completionHandler returns a handler for dynamic shell completion requests.
// It queries the PagerDuty API for resources matching the requested completion
// kind (e.g. "team", "service") and prints matching IDs to stdout.
//
// Dynamic completions require a valid API token (PDC_TOKEN env var or OS
// keyring). Each lookup enforces a 5-second timeout to keep tab completion
// responsive. Results respect the team and service filters from config.
//
// Fish shell natively parses tab-separated "ID\tDescription" lines, so the
// handler includes descriptions when the requesting shell is fish. Other
// shells receive bare IDs until clib's generators learn to handle the format.
func completionHandler(token string, cfg *config.Config, opts ...api.Option) complete.Handler {
	// Build filter slices from config. These scope completions to the
	// user's configured team/service so results stay relevant.
	var teamIDs, serviceIDs []string
	if cfg != nil && cfg.Team != "" {
		teamIDs = []string{cfg.Team}
	}
	if cfg != nil && cfg.Service != "" {
		serviceIDs = []string{cfg.Service}
	}

	return func(shell, kind string, args []string) {
		if token == "" {
			return
		}
		client := api.NewClient(token, opts...)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// printCompletion prints a completion candidate. Fish receives
		// "ID\tDescription"; other shells receive bare IDs.
		printCompletion := func(id, desc string) {
			if shell == "fish" && desc != "" {
				_, _ = fmt.Printf("%s\t%s\n", id, desc)
			} else {
				_, _ = fmt.Println(id)
			}
		}

		switch kind {
		case "team":
			teams, err := client.ListTeams(ctx, api.ListTeamsOpts{})
			if err != nil {
				return
			}
			for _, t := range teams {
				printCompletion(t.ID, t.Name)
			}
		case "service":
			services, err := client.ListServices(ctx, api.ListServicesOpts{TeamIDs: teamIDs})
			if err != nil {
				return
			}
			for _, s := range services {
				printCompletion(s.ID, s.Name)
			}
		case "user":
			users, err := client.ListUsers(ctx, api.ListUsersOpts{TeamIDs: teamIDs})
			if err != nil {
				return
			}
			for _, u := range users {
				printCompletion(u.ID, u.Name)
			}
		case "schedule":
			schedules, err := client.ListSchedules(ctx, api.ListSchedulesOpts{})
			if err != nil {
				return
			}
			for _, s := range schedules {
				printCompletion(s.ID, s.Name)
			}
		case "incident":
			incidents, err := client.ListIncidents(ctx, api.ListIncidentsOpts{
				Statuses:   []string{"triggered", "acknowledged"},
				TeamIDs:    teamIDs,
				ServiceIDs: serviceIDs,
			})
			if err != nil {
				return
			}
			for _, inc := range incidents {
				printCompletion(inc.ID, inc.Title)
			}
		case "alert":
			if len(args) == 0 {
				return
			}
			alerts, err := client.ListIncidentAlerts(ctx, args[0])
			if err != nil {
				return
			}
			for _, a := range alerts {
				if a.Status != "resolved" {
					printCompletion(a.ID, a.Summary)
				}
			}
		case "escalation_policy":
			policies, err := client.ListEscalationPolicies(ctx, api.ListEscalationPoliciesOpts{TeamIDs: teamIDs})
			if err != nil {
				return
			}
			for _, p := range policies {
				printCompletion(p.ID, p.Name)
			}
		case "maintenance_window":
			windows, err := client.ListMaintenanceWindows(ctx, api.ListMaintenanceWindowsOpts{
				TeamIDs:    teamIDs,
				ServiceIDs: serviceIDs,
			})
			if err != nil {
				return
			}
			for _, w := range windows {
				printCompletion(w.ID, w.Description)
			}
		case "urgency":
			_, _ = fmt.Println("high")
			_, _ = fmt.Println("low")
		default:
			return
		}
	}
}
