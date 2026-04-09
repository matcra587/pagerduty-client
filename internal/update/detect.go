// Package update implements self-update detection, version checking and
// upgrade mechanics for pdc.
package update

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

const modulePath = "github.com/matcra587/pagerduty-client"

// InstallMethod describes how pdc was installed.
type InstallMethod int

const (
	// Binary is a standalone binary from a GitHub release or manual build.
	Binary InstallMethod = iota
	// Homebrew was installed via Homebrew.
	Homebrew
	// GoInstall was installed via `go install`.
	GoInstall
)

// String returns a human-readable label for the install method.
func (m InstallMethod) String() string {
	switch m {
	case Homebrew:
		return "homebrew"
	case GoInstall:
		return "go install"
	default:
		return "binary"
	}
}

// homebrewPrefixes are path prefixes that indicate a Homebrew installation.
var homebrewPrefixes = []string{
	"/opt/homebrew/",
	"/usr/local/Cellar/",
	"/home/linuxbrew/.linuxbrew/",
}

// DetectMethod determines how pdc was installed by inspecting the
// resolved executable path and embedded build information.
func DetectMethod() InstallMethod {
	exe, err := os.Executable()
	if err != nil {
		return Binary
	}

	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}

	var goMod string
	if info, ok := debug.ReadBuildInfo(); ok {
		goMod = info.Path
	}

	return DetectMethodFromPath(resolved, goMod)
}

// IsHomebrewHEAD returns true if the current binary is a Homebrew
// HEAD install. Only meaningful when DetectMethod returns Homebrew.
func IsHomebrewHEAD() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return false
	}
	return IsHomebrewHEADFromPath(resolved)
}

// IsHomebrewHEADFromPath is the testable core of HEAD detection.
// Homebrew builds HEAD formulae into directories named HEAD-<hash>
// under the Cellar (e.g. /opt/homebrew/Cellar/<name>/HEAD-abc1234/).
func IsHomebrewHEADFromPath(binPath string) bool {
	isHomebrew := false
	for _, prefix := range homebrewPrefixes {
		if strings.HasPrefix(binPath, prefix) {
			isHomebrew = true
			break
		}
	}
	if !isHomebrew {
		return false
	}

	// Find the Cellar segment and extract the version directory.
	// Path layout: <prefix>[/Cellar]/<formula>/<version>/bin/<binary>
	_, rest, found := strings.Cut(binPath, "/Cellar/")
	if !found {
		return false
	}

	parts := strings.SplitN(rest, "/", 3)
	return len(parts) >= 2 && strings.HasPrefix(parts[1], "HEAD-")
}

// DetectMethodFromPath is the testable core of install method detection.
// It takes a resolved binary path and the module path from build info.
func DetectMethodFromPath(binPath, goMod string) InstallMethod {
	for _, prefix := range homebrewPrefixes {
		if strings.HasPrefix(binPath, prefix) {
			return Homebrew
		}
	}

	if goMod == modulePath {
		return GoInstall
	}

	return Binary
}
