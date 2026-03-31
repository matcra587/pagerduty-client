package cmd

import (
	"fmt"
	"os"

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
		agentFlag, _ := cmd.Root().PersistentFlags().GetBool("agent")
		det := agent.DetectWithFlag(agentFlag)
		info := version.Info()

		if det.Active {
			return output.RenderAgentJSON(os.Stdout, "version", output.ResourceNone, info, nil, nil)
		}

		_, _ = fmt.Fprintln(os.Stdout, info.String())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
