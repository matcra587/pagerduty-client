package tui

import (
	"fmt"

	"charm.land/huh/v2"
	"github.com/matcra587/pagerduty-client/internal/update"
)

// UpdateChoice represents the user's decision on the update prompt.
type UpdateChoice int

const (
	// UpdateNow means the user chose to update immediately.
	UpdateNow UpdateChoice = iota
	// UpdateSkip means the user skipped for this session.
	UpdateSkip
	// UpdateDismiss means the user dismissed until the next version.
	UpdateDismiss
)

// RunUpdatePrompt shows an interactive update prompt and returns the
// user's choice. Returns UpdateSkip if the prompt is cancelled.
// For dev channel, current and latest are commit SHAs.
func RunUpdatePrompt(current, latest string, ch update.Channel) (UpdateChoice, error) {
	var (
		title       string
		description string
		choice      string
	)

	switch ch {
	case update.ChannelDev:
		title = fmt.Sprintf("Dev build is behind main (%s -> %s)", current, latest)
		description = fmt.Sprintf("https://github.com/matcra587/pagerduty-client/compare/%s...%s", current, latest)
	default:
		title = fmt.Sprintf("Update available: v%s -> v%s", current, latest)
		description = "https://github.com/matcra587/pagerduty-client/releases/tag/v" + latest
	}

	err := huh.NewSelect[string]().
		Title(title).
		Description(description).
		Options(
			huh.NewOption("Update now", "update"),
			huh.NewOption("Skip", "skip"),
			huh.NewOption("Skip until next version", "dismiss"),
		).
		Value(&choice).
		Run()
	if err != nil {
		return UpdateSkip, err
	}

	switch choice {
	case "update":
		return UpdateNow, nil
	case "dismiss":
		return UpdateDismiss, nil
	default:
		return UpdateSkip, nil
	}
}
