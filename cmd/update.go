package cmd

import (
	"os"

	clib "github.com/gechr/clib/cli/cobra"
	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"up"},
	Short:   "Update pdc to the latest version",
	Long:    "Detect the install method and update pdc to the latest tagged release (stable) or latest commit on main (dev).",
	Example: `# Update pdc to the latest stable release
$ pdc update

# Update to the latest dev build
$ pdc update --channel dev`,
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
		channelStr, _ := cmd.Flags().GetString("channel")

		// Fall back to env, then config file. The update command
		// bypasses root PersistentPreRunE so config is not on
		// context — load it directly.
		if channelStr == "" {
			channelStr = os.Getenv("PDC_UPDATE_CHANNEL")
		}
		if channelStr == "" {
			if cfg, err := config.Load(); err == nil {
				channelStr = cfg.UpdateChannel
			}
		}

		ch, err := update.ParseChannel(channelStr)
		if err != nil {
			return err
		}

		return update.Run(cmd.Context(), ch)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().String("channel", "", "Update channel")
	clib.Extend(updateCmd.Flags().Lookup("channel"), clib.FlagExtra{
		Group:       "Action",
		Placeholder: "CHANNEL",
		Terse:       "update channel",
		Enum:        []string{"stable", "dev"},
		EnumTerse:   []string{"latest tagged release", "latest commit on main"},
		EnumDefault: "stable",
	})
}
