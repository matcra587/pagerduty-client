package update

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/version"
)

// Run performs the update for the detected install method.
func Run(ctx context.Context) error {
	method := DetectMethod()
	clog.Info().Str("method", method.String()).Msg("detected install method")

	latest, err := FetchLatestVersion(ctx, "")
	if err != nil {
		return err
	}

	if !IsNewer(version.Version, latest) {
		clog.Info().Str("version", version.Version).Msg("already up to date")
		return nil
	}

	switch method {
	case Homebrew:
		return runBrewUpgrade(ctx, latest)
	case GoInstall:
		return runGoInstall(ctx, latest)
	case Binary:
		return runSelfReplace(ctx, latest)
	}
	return nil
}

func runBrewUpgrade(ctx context.Context, latest string) error {
	brewPath, err := exec.LookPath("brew")
	if err != nil {
		return errors.New("brew not found on PATH: install manually from https://github.com/matcra587/pagerduty-client/releases")
	}

	return clog.Spinner("Updating via Homebrew").
		Str("version", latest).
		Elapsed("elapsed").
		Wait(ctx, func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, brewPath, "upgrade", "matcra587/tap/pagerduty-client") //nolint:gosec // brewPath validated by LookPath above
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}).
		Msg("Updated")
}

func runGoInstall(ctx context.Context, latest string) error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return errors.New("go not found on PATH: install manually from https://github.com/matcra587/pagerduty-client/releases")
	}

	target := modulePath + "/cmd/pdc@latest"

	return clog.Spinner("Updating via go install").
		Str("version", latest).
		Elapsed("elapsed").
		Wait(ctx, func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, goPath, "install", target) //nolint:gosec // goPath validated by LookPath above
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}).
		Msg("Updated")
}

func runSelfReplace(ctx context.Context, latest string) error {
	return selfReplace(ctx, latest)
}
