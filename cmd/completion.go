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
