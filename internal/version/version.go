// Package version exposes build-time version information injected via ldflags.
package version

import (
	"fmt"
	"runtime/debug"
)

//nolint:gochecknoglobals
var (
	// Version is the release version, set via ldflags at build time.
	Version = "dev"
	// Commit is the git commit hash, set via ldflags at build time.
	Commit = "unknown"
	// Branch is the git branch, set via ldflags at build time.
	Branch = "unknown"
	// BuildTime is the build timestamp, set via ldflags at build time.
	BuildTime = "unknown"
	// BuildBy is the build system identifier, set via ldflags at build time.
	BuildBy = "unknown"
)

// BuildInfo holds version metadata populated at build time.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Branch    string `json:"branch"`
	BuildTime string `json:"build_time"`
	BuildBy   string `json:"build_by"`
}

// Info returns the current build information.
func Info() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		Branch:    Branch,
		BuildTime: BuildTime,
		BuildBy:   BuildBy,
	}
}

// String returns a plain multi-line summary of the build information.
func (b BuildInfo) String() string {
	return fmt.Sprintf(`pdc %s
  commit:   %s
  branch:   %s
  built:    %s
  built by: %s`, b.Version, b.Commit, b.Branch, b.BuildTime, b.BuildBy)
}

// ResolvedCommit returns the build commit hash. It prefers the
// ldflags-injected Commit value; when that is "unknown" (e.g.
// go install @main), it falls back to vcs.revision from build info.
func ResolvedCommit() string {
	if Commit != "unknown" {
		return Commit
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value[:min(12, len(s.Value))]
		}
	}
	return "unknown"
}
