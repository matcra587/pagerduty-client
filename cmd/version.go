package cmd

import (
	"fmt"
	"os"

	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/matcra587/pagerduty-client/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the pdc version, commit, branch and build metadata.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		info := version.Info()
		det := AgentFromContext(cmd)

		if det.Active {
			return output.RenderAgentJSON(os.Stdout, "version", info, nil, nil)
		}

		_, _ = fmt.Fprintln(os.Stdout, info.String())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
