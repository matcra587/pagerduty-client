package tui

import (
	"fmt"

	"charm.land/huh/v2"
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
func RunUpdatePrompt(current, latest string) (UpdateChoice, error) {
	var choice string

	err := huh.NewSelect[string]().
		Title(fmt.Sprintf("Update available: v%s -> v%s", current, latest)).
		Description("https://github.com/matcra587/pagerduty-client/releases/tag/v"+latest).
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
