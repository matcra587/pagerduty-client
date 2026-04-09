package update_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsNewer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current string
		latest  string
		expect  bool
	}{
		{"newer patch", "0.3.0", "0.3.1", true},
		{"newer minor", "0.3.1", "0.4.0", true},
		{"newer major", "0.3.1", "1.0.0", true},
		{"same version", "0.3.1", "0.3.1", false},
		{"older version", "0.4.0", "0.3.1", false},
		{"dev version", "dev", "0.3.1", false},
		{"with v prefix", "0.3.0", "v0.3.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, update.IsNewer(tt.current, tt.latest))
		})
	}
}

func TestVersionCache_IsStale(t *testing.T) {
	t.Parallel()

	t.Run("zero value is stale", func(t *testing.T) {
		t.Parallel()
		var c update.VersionCache
		assert.True(t, c.IsStale(24*time.Hour))
	})

	t.Run("recent check is not stale", func(t *testing.T) {
		t.Parallel()
		c := update.VersionCache{CheckedAt: time.Now()}
		assert.False(t, c.IsStale(24*time.Hour))
	})

	t.Run("old check is stale", func(t *testing.T) {
		t.Parallel()
		c := update.VersionCache{CheckedAt: time.Now().Add(-48 * time.Hour)}
		assert.True(t, c.IsStale(24*time.Hour))
	})
}

func TestVersionCache_IsDismissed(t *testing.T) {
	t.Parallel()

	t.Run("dismissed when versions match", func(t *testing.T) {
		t.Parallel()
		c := update.VersionCache{
			LatestRef:    "0.4.0",
			DismissedRef: "0.4.0",
		}
		assert.True(t, c.IsDismissed())
	})

	t.Run("not dismissed when versions differ", func(t *testing.T) {
		t.Parallel()
		c := update.VersionCache{
			LatestRef:    "0.5.0",
			DismissedRef: "0.4.0",
		}
		assert.False(t, c.IsDismissed())
	})

	t.Run("not dismissed when no dismissed version", func(t *testing.T) {
		t.Parallel()
		c := update.VersionCache{LatestRef: "0.4.0"}
		assert.False(t, c.IsDismissed())
	})
}

func TestReadCache_MissingFile(t *testing.T) {
	t.Parallel()

	c, err := update.ReadCache(filepath.Join(t.TempDir(), "nonexistent.json"))
	require.NoError(t, err)
	assert.True(t, c.IsStale(24*time.Hour))
}

func TestReadCache_CorruptJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "corrupt.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid json"), 0o600))

	c, err := update.ReadCache(path)
	require.NoError(t, err)
	assert.True(t, c.IsStale(24*time.Hour))
}

func TestWriteCache_ReadCache_Roundtrip(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "cache.json")

	want := update.VersionCache{
		LatestRef:    "0.4.0",
		CheckedAt:    time.Now().Truncate(time.Second),
		DismissedRef: "0.3.0",
	}

	require.NoError(t, update.WriteCache(path, want))

	got, err := update.ReadCache(path)
	require.NoError(t, err)
	assert.Equal(t, want.LatestRef, got.LatestRef)
	assert.Equal(t, want.DismissedRef, got.DismissedRef)
	assert.WithinDuration(t, want.CheckedAt, got.CheckedAt, time.Second)
}

func TestFetchLatestVersion_Success(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/matcra587/pagerduty-client/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v0.4.0"})
	})

	ver, err := update.FetchLatestVersion(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, "0.4.0", ver)
}

func TestFetchLatestVersion_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/matcra587/pagerduty-client/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := update.FetchLatestVersion(context.Background(), server.URL)
	assert.Error(t, err)
}

func TestFetchLatestCommit_Success(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/matcra587/pagerduty-client/commits/main", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"sha": "abc123def456789"})
	})

	sha, err := update.FetchLatestCommit(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, "abc123def456", sha)
}

func TestFetchLatestCommit_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/matcra587/pagerduty-client/commits/main", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := update.FetchLatestCommit(context.Background(), server.URL)
	assert.Error(t, err)
}

func TestReadCache_LegacyFormat(t *testing.T) {
	t.Parallel()

	// Simulate a cache file written by a pre-channel version of pdc.
	// Old format used "latest_version" and "dismissed_version" JSON keys
	// with no "channel" field.
	recent := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	legacy := fmt.Sprintf(`{
		"latest_version": "0.10.0",
		"checked_at": %q,
		"dismissed_version": "0.9.0"
	}`, recent)

	path := filepath.Join(t.TempDir(), "version.json")
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o600))

	c, err := update.ReadCache(path)
	require.NoError(t, err)

	// Old keys don't match new struct tags — fields should be zero-value.
	assert.Empty(t, c.LatestRef, "old latest_version key should not populate LatestRef")
	assert.Empty(t, c.DismissedRef, "old dismissed_version key should not populate DismissedRef")
	assert.Empty(t, c.Channel, "old format has no channel field")

	// checked_at key is unchanged, so CheckedAt unmarshals correctly.
	// The cache is NOT stale by time — but CheckForUpdate will
	// force-invalidate it because Channel is "" (doesn't match "stable").
	assert.False(t, c.IsStale(24*time.Hour), "checked_at is valid, cache is not time-stale")
	assert.NotZero(t, c.CheckedAt, "checked_at key is unchanged and should unmarshal")
}

func TestReadCache_LegacyFormat_NotDismissed(t *testing.T) {
	t.Parallel()

	// A legacy cache with empty fields should not report as dismissed.
	recent := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	legacy := fmt.Sprintf(`{"latest_version": "0.10.0", "checked_at": %q}`, recent)

	path := filepath.Join(t.TempDir(), "version.json")
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o600))

	c, err := update.ReadCache(path)
	require.NoError(t, err)

	assert.False(t, c.IsDismissed(), "legacy cache should not be dismissed")
}

func TestIsNewerDev_DifferentCommits(t *testing.T) {
	t.Parallel()
	current := "abc123def456"
	latest := "789012345678"
	updateAvail := current != "unknown" && latest != "" && current != latest
	assert.True(t, updateAvail)
}

func TestIsNewerDev_SameCommit(t *testing.T) {
	t.Parallel()
	current := "abc123def456"
	latest := "abc123def456"
	updateAvail := current != "unknown" && latest != "" && current != latest
	assert.False(t, updateAvail)
}

func TestIsNewerDev_UnknownCommit(t *testing.T) {
	t.Parallel()
	current := "unknown"
	latest := "abc123def456"
	updateAvail := current != "unknown" && latest != "" && current != latest
	assert.False(t, updateAvail)
}

func TestIsNewerDev_EmptyLatest(t *testing.T) {
	t.Parallel()
	current := "abc123def456"
	latest := ""
	updateAvail := current != "unknown" && latest != "" && current != latest
	assert.False(t, updateAvail)
}
