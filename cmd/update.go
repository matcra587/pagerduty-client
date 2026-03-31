package cmd

import (
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update pdc to the latest version",
	Long:  "Detect the install method and update pdc to the latest tagged release.",
	Example: `# Update pdc to the latest version
$ pdc update`,
	Args: cobra.NoArgs,
	// Bypass token resolution: update does not need PagerDuty credentials.
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		pf := cmd.Root().PersistentFlags()
		if debug, _ := pf.GetBool("debug"); debug {
			clog.SetVerbose(true)
		}
		if colorMode, _ := pf.GetString("color"); colorMode != "" {
			switch colorMode {
			case "always":
				clog.SetColorMode(clog.ColorAlways)
			case "never":
				clog.SetColorMode(clog.ColorNever)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		return update.Run(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
