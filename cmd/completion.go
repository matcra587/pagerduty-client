package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gechr/clib/complete"
	"github.com/matcra587/pagerduty-client/internal/api"
)

// completionHandler returns a handler for dynamic shell completion requests.
// It queries the PagerDuty API for resources matching the requested completion
// kind (e.g. "team", "service") and prints matching IDs to stdout.
func completionHandler(token string, opts ...api.Option) complete.Handler {
	return func(_, kind string, args []string) {
		if token == "" {
			return
		}
		client := api.NewClient(token, opts...)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var names []string
		switch kind {
		case "team":
			teams, err := client.ListTeams(ctx, api.ListTeamsOpts{})
			if err != nil {
				return
			}
			for _, t := range teams {
				names = append(names, t.ID)
			}
		case "service":
			services, err := client.ListServices(ctx, api.ListServicesOpts{})
			if err != nil {
				return
			}
			for _, s := range services {
				names = append(names, s.ID)
			}
		case "user":
			users, err := client.ListUsers(ctx, api.ListUsersOpts{})
			if err != nil {
				return
			}
			for _, u := range users {
				names = append(names, u.ID)
			}
		case "schedule":
			schedules, err := client.ListSchedules(ctx, api.ListSchedulesOpts{})
			if err != nil {
				return
			}
			for _, s := range schedules {
				names = append(names, s.ID)
			}
		case "incident":
			incidents, err := client.ListIncidents(ctx, api.ListIncidentsOpts{
				Statuses: []string{"triggered", "acknowledged"},
			})
			if err != nil {
				return
			}
			for _, inc := range incidents {
				names = append(names, inc.ID)
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
					names = append(names, a.ID)
				}
			}
		case "urgency":
			names = append(names, "high", "low")
		default:
			return
		}
		for _, n := range names {
			_, _ = fmt.Println(n)
		}
	}
}

// setDynamicArgs configures positional arg completion for every
// subcommand that accepts positional arguments.
//
// TODO: Replace with command-level annotations once clib's Cobra
// bridge parses DynamicArgs (e.g. dynamic-args='incident,alert').
// Today it only reads complete='path' for PathArgs, so we set
// DynamicArgs on the SubSpec tree after clib.Subcommands() builds it.
//
// Completion kinds map to cases in completionHandler:
//
//	"incident" -> list triggered/acknowledged incident IDs
//	"alert"    -> list non-resolved alert IDs for a given incident
//	"urgency"  -> static: "high", "low"
//	"service"  -> list service IDs
//	"user"     -> list user IDs
//	"team"     -> list team IDs
//	"schedule" -> list schedule IDs
func setDynamicArgs(subs []complete.SubSpec) {
	for i := range subs {
		switch subs[i].Name {
		case "incident":
			for j := range subs[i].Subs {
				switch subs[i].Subs[j].Name {
				case "list":
					// No positional args.
				case "resolve-alert":
					// <incident-id> <alert-id>...
					// TODO: clib's zsh generator maps position 2 to dyn_2
					// but not 3+. Variadic alert completion stops after
					// the first alert ID. Track upstream fix.
					subs[i].Subs[j].DynamicArgs = []string{"incident", "alert"}
				case "urgency":
					// <incident-id> <high|low>
					subs[i].Subs[j].DynamicArgs = []string{"incident", "urgency"}
				case "title":
					// <incident-id> <new-title> - title is free text, no completion.
					subs[i].Subs[j].DynamicArgs = []string{"incident"}
				case "note":
					// note add <id>, note list <id>
					for k := range subs[i].Subs[j].Subs {
						subs[i].Subs[j].Subs[k].DynamicArgs = []string{"incident"}
					}
				default:
					// show, ack, resolve, snooze, reassign, merge, log
					// all take <incident-id> as the first positional arg.
					subs[i].Subs[j].DynamicArgs = []string{"incident"}
				}
			}

		case "service":
			for j := range subs[i].Subs {
				if subs[i].Subs[j].Name == "show" {
					subs[i].Subs[j].DynamicArgs = []string{"service"}
				}
			}

		case "user":
			for j := range subs[i].Subs {
				if subs[i].Subs[j].Name == "show" {
					subs[i].Subs[j].DynamicArgs = []string{"user"}
				}
			}

		case "team":
			for j := range subs[i].Subs {
				if subs[i].Subs[j].Name == "show" {
					subs[i].Subs[j].DynamicArgs = []string{"team"}
				}
			}

		case "schedule":
			for j := range subs[i].Subs {
				switch subs[i].Subs[j].Name {
				case "show", "override":
					// <schedule-id>
					subs[i].Subs[j].DynamicArgs = []string{"schedule"}
				}
			}
		}
	}
}
