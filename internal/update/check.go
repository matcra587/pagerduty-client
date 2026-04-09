package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	ghAuth "github.com/cli/go-gh/v2/pkg/auth"
	"github.com/gechr/clog"
	"golang.org/x/mod/semver"
)

const (
	defaultAPIBase = "https://api.github.com"
	repoPath       = "/repos/matcra587/pagerduty-client/releases/latest"
	httpTimeout    = 3 * time.Second
)

// VersionCache holds the result of a GitHub release check.
type VersionCache struct {
	Channel      string    `json:"channel"`
	LatestRef    string    `json:"latest_ref"`
	CheckedAt    time.Time `json:"checked_at"`
	DismissedRef string    `json:"dismissed_ref"`
}

// IsStale returns true if CheckedAt is zero or older than maxAge.
func (c VersionCache) IsStale(maxAge time.Duration) bool {
	if c.CheckedAt.IsZero() {
		return true
	}
	return time.Since(c.CheckedAt) > maxAge
}

// IsDismissed returns true if DismissedRef equals LatestRef.
func (c VersionCache) IsDismissed() bool {
	return c.DismissedRef != "" && c.DismissedRef == c.LatestRef
}

// ReadCache reads a VersionCache from disk. A missing or corrupt file
// returns a zero-value cache (which is stale), not an error.
func ReadCache(path string) (VersionCache, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			clog.Debug().Err(err).Str("path", path).Msg("could not read version cache")
		}
		return VersionCache{}, nil
	}

	var c VersionCache
	if err := json.Unmarshal(data, &c); err != nil {
		return VersionCache{}, nil //nolint:nilerr // corrupt JSON treated as empty cache by design
	}

	return c, nil
}

// WriteCache writes a VersionCache to disk atomically using a temporary
// file and rename. It creates the parent directory if needed.
func WriteCache(path string, c VersionCache) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshalling cache: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".pdc-cache-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("renaming cache file: %w", err)
	}

	return nil
}

// FetchLatestVersion queries the GitHub API for the latest release and
// returns the version string with the "v" prefix stripped. An empty
// baseURL defaults to https://api.github.com.
func FetchLatestVersion(ctx context.Context, baseURL string) (string, error) {
	if baseURL == "" {
		baseURL = defaultAPIBase
	}

	ctx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	url := baseURL + repoPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) //nolint:gosec // baseURL is a constant or test override, not user input
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}

	setGitHubAuth(req)

	client := &http.Client{
		CheckRedirect: stripAuthOnRedirect,
	}

	resp, err := client.Do(req) //nolint:gosec // URL constructed from trusted constant
	if err != nil {
		return "", fmt.Errorf("fetching latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from GitHub API", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decoding release response: %w", err)
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}

// IsNewer returns true if latest is a newer semver than current.
// Returns false if either version is unparseable (e.g. "dev").
func IsNewer(current, latest string) bool {
	// Strip build metadata and dirty suffixes so local builds
	// (e.g. "v0.6.3-dirty", "v0.6.3-2-gabcdef-dirty") are not
	// considered older than the release they are based on.
	current = stripBuildSuffix(current)
	latest = stripBuildSuffix(latest)

	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}

	if !semver.IsValid(current) || !semver.IsValid(latest) {
		return false
	}

	return semver.Compare(current, latest) < 0
}

// stripBuildSuffix removes git-describe noise (commit count, hash,
// dirty flag) from a version string. "0.6.3-2-gabcdef-dirty" becomes
// "0.6.3", "0.6.3-dirty" becomes "0.6.3".
func stripBuildSuffix(v string) string {
	v = strings.TrimSuffix(v, "-dirty")
	// git describe: "v0.6.3-2-gabcdef" - strip "-N-gHASH"
	if i := strings.LastIndex(v, "-g"); i > 0 {
		if j := strings.LastIndex(v[:i], "-"); j > 0 {
			v = v[:j]
		}
	}
	return v
}

// setGitHubAuth sets the Authorization header using go-gh's token
// resolution. This picks up GH_TOKEN, GITHUB_TOKEN and tokens from
// gh auth login (stored in the gh config file).
func setGitHubAuth(req *http.Request) {
	if token, _ := ghAuth.TokenForHost("github.com"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// stripAuthOnRedirect removes the Authorization header when a redirect
// targets a different host. GitHub Releases redirects to a CDN; without
// this the token leaks to the CDN host.
func stripAuthOnRedirect(req *http.Request, via []*http.Request) error {
	if len(via) > 0 && req.URL.Host != via[0].URL.Host {
		req.Header.Del("Authorization")
	}
	return nil
}
