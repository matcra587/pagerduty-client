package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
// whether an update is available. The channel determines whether
// to check the latest release (stable) or latest commit (dev).
func CheckForUpdate(ctx context.Context, ch Channel) CheckResult {
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

	ch = ch.Effective()

	// Invalidate cache when channel changed.
	if cache.Channel != ch.String() {
		cache = VersionCache{}
	}

	if cache.IsStale(cacheTTL) {
		baseURL := os.Getenv("PDC_UPDATE_URL")

		switch ch {
		case ChannelStable:
			latest, err := FetchLatestVersion(ctx, baseURL)
			if err != nil {
				clog.Debug().Err(err).Msg("could not fetch latest version")
				return resultFromCache(cache, ch)
			}
			cache.LatestRef = latest
		case ChannelDev:
			sha, err := FetchLatestCommit(ctx, baseURL)
			if err != nil {
				clog.Debug().Err(err).Msg("could not fetch latest commit")
				return resultFromCache(cache, ch)
			}
			cache.LatestRef = sha
		}

		cache.Channel = ch.String()
		cache.CheckedAt = time.Now()
		if err := WriteCache(cachePath, cache); err != nil {
			clog.Debug().Err(err).Msg("could not write version cache")
		}
	}

	return resultFromCache(cache, ch)
}

func resultFromCache(cache VersionCache, ch Channel) CheckResult {
	switch ch {
	case ChannelDev:
		current := version.ResolvedCommit()
		return CheckResult{
			LatestRef:   cache.LatestRef,
			Channel:     ch,
			UpdateAvail: current != "unknown" && cache.LatestRef != "" && !commitsMatch(current, cache.LatestRef),
			Dismissed:   cache.IsDismissed(),
		}
	default:
		current := version.Version
		return CheckResult{
			LatestRef:   cache.LatestRef,
			Channel:     ch,
			UpdateAvail: IsNewer(current, cache.LatestRef),
			Dismissed:   cache.IsDismissed(),
		}
	}
}

// NotifyCLI prints a one-line update banner to stderr.
func NotifyCLI(result CheckResult) {
	if !result.UpdateAvail || result.Dismissed {
		return
	}

	switch result.Channel {
	case ChannelDev:
		current := version.ResolvedCommit()
		fmt.Fprintf(os.Stderr, "\nCurrent build is behind latest dev build (%s -> %s)\nRun \"pdc update\" to update.\n\n",
			current, result.LatestRef)
	default:
		fmt.Fprintf(os.Stderr, "\nA new version of pdc is available: v%s -> v%s\nRun \"pdc update\" to update.\n\n",
			version.Version, result.LatestRef)
	}
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

// commitsMatch returns true if two commit SHAs refer to the same
// commit. Handles different truncation lengths by comparing on the
// shorter of the two (e.g. 7-char ldflags vs 12-char API response).
func commitsMatch(a, b string) bool {
	n := min(len(a), len(b))
	if n == 0 {
		return false
	}
	return strings.EqualFold(a[:n], b[:n])
}
