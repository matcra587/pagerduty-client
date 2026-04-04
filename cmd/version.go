package cmd

import (
	"fmt"
	"os"

	"github.com/gechr/clib/terminal"
	"github.com/gechr/clib/theme"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/matcra587/pagerduty-client/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the pdc version, commit, branch and build metadata.",
	Example: `# Print version information
$ pdc version`,
	Args: cobra.NoArgs,
	// Suppress root PersistentPreRunE: version does not require a token.
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error { return nil },
	RunE: func(cmd *cobra.Command, _ []string) error {
		pf := cmd.Root().PersistentFlags()
		agentFlag, _ := pf.GetBool("agent")
		det := agent.DetectWithFlag(agentFlag)
		info := version.Info()

		if det.Active {
			return output.RenderAgentJSON(os.Stdout, "version", output.ResourceNone, info, nil, nil)
		}

		w := os.Stdout
		if terminal.Is(w) {
			th := theme.New()
			fmt.Fprintf(w, "%s %s\n", th.Bold.Render("pdc"), th.Green.Render(info.Version))
			fmt.Fprintf(w, "  %s  %s\n", th.Dim.Render("commit:"), info.Commit)
			fmt.Fprintf(w, "  %s  %s\n", th.Dim.Render("branch:"), info.Branch)
			fmt.Fprintf(w, "  %s   %s\n", th.Dim.Render("built:"), info.BuildTime)
			fmt.Fprintf(w, "  %s %s\n", th.Dim.Render("built by:"), info.BuildBy)
		} else {
			fmt.Fprintln(w, info.String())
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
