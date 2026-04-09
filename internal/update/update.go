package update

import (
	"context"
	"errors"
	"os/exec"

	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/version"
)

// ValidateMethodChannel checks whether the install method
// supports the requested channel. Returns an error with
// remediation guidance for unsupported combinations.
func ValidateMethodChannel(method InstallMethod, isHEAD bool, ch Channel) error {
	switch method {
	case Homebrew:
		switch {
		case ch == ChannelStable && isHEAD:
			return errors.New("current install is Homebrew (HEAD) — switch to stable with: brew uninstall pagerduty-client && brew install matcra587/tap/pagerduty-client")
		case ch == ChannelDev && !isHEAD:
			return errors.New("current install is Homebrew (stable) — switch to dev with: brew uninstall pagerduty-client && brew install --HEAD matcra587/tap/pagerduty-client")
		}
	case Binary:
		if ch == ChannelDev {
			return errors.New("dev channel is not supported for standalone binaries — install via Homebrew (--HEAD) or go install")
		}
	}
	return nil
}

// Run performs the update for the detected install method and channel.
func Run(ctx context.Context, ch Channel) error {
	ch = ch.Effective()
	method := DetectMethod()
	clog.Info().Str("method", method.String()).Str("channel", ch.String()).Msg("detected install method")

	switch method {
	case Homebrew:
		return runHomebrew(ctx, ch)
	case GoInstall:
		return runGoInstall(ctx, ch)
	case Binary:
		return runBinary(ctx, ch)
	}
	return nil
}

func runHomebrew(ctx context.Context, ch Channel) error {
	if err := ValidateMethodChannel(Homebrew, IsHomebrewHEAD(), ch); err != nil {
		return err
	}

	brewPath, err := exec.LookPath("brew")
	if err != nil {
		return errors.New("brew not found on PATH: install manually from https://github.com/matcra587/pagerduty-client/releases")
	}

	args := []string{"upgrade"}
	label := "Updating via Homebrew"
	if ch == ChannelDev {
		args = append(args, "--fetch-HEAD")
		label = "Updating via Homebrew (HEAD)"
	}
	args = append(args, "matcra587/tap/pagerduty-client")

	return clog.Spinner(label).
		Elapsed("elapsed").
		Wait(ctx, func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, brewPath, args...) //nolint:gosec // brewPath validated by LookPath above
			return cmd.Run()
		}).
		Msg("Updated")
}

func runGoInstall(ctx context.Context, ch Channel) error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return errors.New("go not found on PATH: install manually from https://github.com/matcra587/pagerduty-client/releases")
	}

	ref := "latest"
	label := "latest release"
	if ch == ChannelDev {
		ref = "main"
		label = "main branch"
	}

	target := modulePath + "/cmd/pdc@" + ref

	return clog.Spinner("Updating via go install").
		Str("target", label).
		Elapsed("elapsed").
		Wait(ctx, func(ctx context.Context) error {
			cmd := exec.CommandContext(ctx, goPath, "install", target) //nolint:gosec // goPath validated by LookPath above
			return cmd.Run()
		}).
		Msg("Updated")
}

func runBinary(ctx context.Context, ch Channel) error {
	if err := ValidateMethodChannel(Binary, false, ch); err != nil {
		return err
	}

	latest, err := FetchLatestVersion(ctx, "")
	if err != nil {
		return err
	}

	if !IsNewer(version.Version, latest) {
		clog.Info().Str("version", version.Version).Msg("already up to date")
		return nil
	}

	return selfReplace(ctx, latest)
}
