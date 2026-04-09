package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gechr/clog"
	"github.com/matcra587/pagerduty-client/internal/dirs"
	"github.com/matcra587/pagerduty-client/internal/version"
)

const (
	cacheFile = "version.json"
	cacheTTL  = 24 * time.Hour
)

// CheckResult holds the outcome of a version check.
type CheckResult struct {
	LatestRef   string
	Channel     Channel
	UpdateAvail bool
	Dismissed   bool
}

// CheckForUpdate reads the cache, refreshes if stale and returns
// whether an update is available.
func CheckForUpdate(ctx context.Context) CheckResult {
	cacheDir, err := dirs.PdcCacheDir()
	if err != nil {
		clog.Debug().Err(err).Msg("could not resolve cache directory")
		return CheckResult{}
	}

	cachePath := filepath.Join(cacheDir, cacheFile)
	cache, err := ReadCache(cachePath)
	if err != nil {
		clog.Debug().Err(err).Msg("could not read version cache")
		return CheckResult{}
	}

	if cache.IsStale(cacheTTL) {
		baseURL := os.Getenv("PDC_UPDATE_URL")
		latest, err := FetchLatestVersion(ctx, baseURL)
		if err != nil {
			clog.Debug().Err(err).Msg("could not fetch latest version")
			return resultFromCache(cache)
		}
		cache.LatestRef = latest
		cache.CheckedAt = time.Now()
		if err := WriteCache(cachePath, cache); err != nil {
			clog.Debug().Err(err).Msg("could not write version cache")
		}
	}

	return resultFromCache(cache)
}

func resultFromCache(cache VersionCache) CheckResult {
	current := version.Version
	return CheckResult{
		LatestRef:   cache.LatestRef,
		Channel:     ChannelStable,
		UpdateAvail: IsNewer(current, cache.LatestRef),
		Dismissed:   cache.IsDismissed(),
	}
}

// NotifyCLI prints a one-line update banner to stderr.
func NotifyCLI(result CheckResult) {
	if !result.UpdateAvail || result.Dismissed {
		return
	}
	fmt.Fprintf(os.Stderr, "\nA new version of pdc is available: v%s -> v%s\nRun \"pdc update\" to update.\n\n",
		version.Version, result.LatestRef)
}

// ShouldCheck returns false when update checks should be skipped.
func ShouldCheck(agentMode, isTTY bool) bool {
	return os.Getenv("PDC_NO_UPDATE_CHECK") == "" && !agentMode && isTTY
}

// DismissVersion writes the dismissed version to the cache so the
// notification is suppressed until a newer version appears.
func DismissVersion(ver string) {
	cacheDir, err := dirs.PdcCacheDir()
	if err != nil {
		return
	}
	cachePath := filepath.Join(cacheDir, cacheFile)
	cache, _ := ReadCache(cachePath)
	cache.DismissedRef = ver
	_ = WriteCache(cachePath, cache)
}
