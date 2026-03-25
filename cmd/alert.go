package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/PagerDuty/go-pagerduty"
	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clib/terminal"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/spf13/cobra"
)

var alertCmd = &cobra.Command{
	Use:     "alert",
	Short:   "View PagerDuty alerts",
	Long:    "List and inspect alerts associated with a PagerDuty incident.",
	GroupID: "resources",
}

var alertListCmd = &cobra.Command{
	Use:   "list",
	Short: "List alerts for an incident",
	RunE: func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		incidentID, _ := cmd.Flags().GetString("incident")
		if incidentID == "" {
			return errors.New("--incident is required")
		}

		alerts, err := client.ListIncidentAlerts(ctx, incidentID)
		if err != nil {
			return fmt.Errorf("listing alerts: %w", err)
		}
		clog.Debug().Elapsed("duration").Int("count", len(alerts)).Msg("listed alerts")

		headers, rows := alertRows(alerts)

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			meta := agent.Metadata{Total: len(alerts)}
			return output.RenderAgentJSON(os.Stdout, "alert list", output.ProjectAlertsForAgent(alerts), &meta, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, alerts, isTTY)
		default:
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

var alertShowCmd = &cobra.Command{
	Use:   "show <alert-id>",
	Short: "Show alert details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := ClientFromContext(cmd)
		cfg := ConfigFromContext(cmd)
		det := AgentFromContext(cmd)

		incidentID, _ := cmd.Flags().GetString("incident")
		if incidentID == "" {
			return errors.New("--incident is required")
		}

		alert, err := client.GetIncidentAlert(ctx, incidentID, args[0])
		if err != nil {
			return fmt.Errorf("getting alert: %w", err)
		}
		clog.Debug().Elapsed("duration").Str("id", args[0]).Msg("fetched alert")

		isTTY := terminal.Is(os.Stdout)
		format := output.DetectFormat(output.FormatOpts{
			AgentMode: det.Active,
			Format:    cfg.Format,
			IsTTY:     isTTY,
		})

		switch format {
		case output.FormatAgentJSON:
			return output.RenderAgentJSON(os.Stdout, "alert show", output.ProjectAlertForAgent(*alert), nil, nil)
		case output.FormatJSON:
			return output.RenderJSON(os.Stdout, alert, isTTY)
		default:
			headers := []string{"Field", "Value"}
			rows := [][]string{
				{"ID", alert.ID},
				{"Status", alert.Status},
				{"Severity", alert.Severity},
				{"Summary", alert.Summary},
				{"Created", alert.CreatedAt},
			}
			return output.RenderTable(os.Stdout, headers, rows, isTTY)
		}
	},
}

// TODO: Port alertBodyCmd once internal/integration package is available (TUI worktree).
// The "alert body" subcommand uses integration.Detect to identify the alert source
// and extract structured fields from the raw alert body payload.

func alertRows(alerts []pagerduty.IncidentAlert) ([]string, [][]string) {
	headers := []string{"ID", "Status", "Severity", "Summary", "Created"}
	rows := make([][]string, len(alerts))
	for i, a := range alerts {
		rows[i] = []string{
			a.ID,
			a.Status,
			a.Severity,
			a.Summary,
			a.CreatedAt,
		}
	}
	return headers, rows
}

func init() {
	rootCmd.AddCommand(alertCmd)
	alertCmd.AddCommand(alertListCmd)
	alertCmd.AddCommand(alertShowCmd)

	alertListCmd.Flags().String("incident", "", "Incident ID (required)")
	clib.Extend(alertListCmd.Flags().Lookup("incident"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Terse:       "incident ID",
	})

	alertShowCmd.Flags().String("incident", "", "Incident ID (required)")
	clib.Extend(alertShowCmd.Flags().Lookup("incident"), clib.FlagExtra{
		Group:       "Filters",
		Placeholder: "ID",
		Terse:       "incident ID",
	})
}
