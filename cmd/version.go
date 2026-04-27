package cmd

import (
	"fmt"
	"os"

	"github.com/gechr/x/terminal"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/output"
	pdctheme "github.com/matcra587/pagerduty-client/internal/tui/theme"
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
		info := version.Info()
		w := cmd.OutOrStdout()

		if isAgentMode(cmd) {
			return output.RenderAgentJSON(w, "version", compact.ResourceNone, info, nil, nil)
		}

		if terminal.Is(os.Stdout) {
			var themeName string
			if cfg, err := config.Load(); err == nil {
				themeName = cfg.UI.Theme
			}
			th := pdctheme.Resolve(themeName)
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
