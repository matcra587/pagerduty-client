package update_test

import (
	"context"
	"encoding/json"
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
