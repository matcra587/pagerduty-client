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
