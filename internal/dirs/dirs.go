// Package dirs provides platform-aware directory helpers for pagerduty-client.
//
// On macOS, os.UserConfigDir and os.UserCacheDir return ~/Library paths.
// This package overrides those to use ~/.config and ~/.cache (honouring
// XDG overrides) so the CLI behaves consistently across platforms.
package dirs

import (
	"os"
	"path/filepath"
	"runtime"
)

const appName = "pagerduty-client"

// PdcConfigDir returns the configuration directory for pagerduty-client.
//
// Resolution:
//   - $XDG_CONFIG_HOME/pagerduty-client if set (all platforms)
//   - ~/.config/pagerduty-client on macOS (override Library path)
//   - os.UserConfigDir()/pagerduty-client elsewhere (honours XDG on Linux)
func PdcConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}

	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", appName), nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}

// PdcCacheDir returns the cache directory for pagerduty-client.
//
// Resolution:
//   - $XDG_CACHE_HOME/pagerduty-client if set (all platforms)
//   - ~/.cache/pagerduty-client on macOS (override Library path)
//   - os.UserCacheDir()/pagerduty-client elsewhere (honours XDG on Linux)
func PdcCacheDir() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}

	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".cache", appName), nil
	}

	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appName), nil
}
