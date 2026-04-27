package cmd

import (
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/gechr/x/terminal"
	"github.com/matcra587/pagerduty-client/internal/resolve"
)

// resolveOrPick takes the result of a Resolver method and returns a
// single ID. On multiple matches, prompts with huh.NewSelect on TTY
// when interactive is true, or returns an error listing matches.
func resolveOrPick(interactive bool, id string, matches []resolve.Match, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if id != "" {
		return id, nil
	}

	// Multiple matches.
	if !interactive || !terminal.Is(os.Stdout) {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = fmt.Sprintf("%s (%s)", m.Name, m.ID)
		}
		return "", fmt.Errorf("multiple matches: %s", strings.Join(names, ", "))
	}

	// Interactive picker.
	options := make([]huh.Option[string], len(matches))
	for i, m := range matches {
		options[i] = huh.NewOption(fmt.Sprintf("%s (%s)", m.Name, m.ID), m.ID)
	}

	var picked string
	err = huh.NewSelect[string]().
		Title("Multiple matches").
		Options(options...).
		Value(&picked).
		Run()
	if err != nil {
		return "", fmt.Errorf("selection cancelled: %w", err)
	}
	return picked, nil
}

// resolveSlice resolves each input in a slice, returning resolved IDs.
func resolveSlice(interactive bool, inputs []string, resolveFn func(string) (string, []resolve.Match, error)) ([]string, error) {
	if len(inputs) == 0 {
		return inputs, nil
	}
	resolved := make([]string, len(inputs))
	for i, input := range inputs {
		rid, matches, fnErr := resolveFn(input)
		id, err := resolveOrPick(interactive, rid, matches, fnErr)
		if err != nil {
			return nil, err
		}
		resolved[i] = id
	}
	return resolved, nil
}
