package dirs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/dirs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPdcConfigDir(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name          string
		xdgConfigHome string
		want          string
		skipNonDarwin bool
		skipNonLinux  bool
	}{
		{
			name:          "xdg override",
			xdgConfigHome: "/tmp/xdgtest",
			want:          "/tmp/xdgtest/pagerduty-client",
		},
		{
			name:          "darwin fallback",
			xdgConfigHome: "",
			want:          filepath.Join(home, ".config", "pagerduty-client"),
			skipNonDarwin: true,
		},
		{
			name:          "linux fallback",
			xdgConfigHome: "",
			want:          filepath.Join(home, ".config", "pagerduty-client"),
			skipNonLinux:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipNonDarwin && runtime.GOOS != "darwin" {
				t.Skip("darwin-only test")
			}
			if tt.skipNonLinux && runtime.GOOS != "linux" {
				t.Skip("linux-only test")
			}

			t.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)

			got, err := dirs.PdcConfigDir()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPdcCacheDir(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name          string
		xdgCacheHome  string
		want          string
		skipNonDarwin bool
		skipNonLinux  bool
	}{
		{
			name:         "xdg override",
			xdgCacheHome: "/tmp/xdgcache",
			want:         "/tmp/xdgcache/pagerduty-client",
		},
		{
			name:          "darwin fallback",
			xdgCacheHome:  "",
			want:          filepath.Join(home, ".cache", "pagerduty-client"),
			skipNonDarwin: true,
		},
		{
			name:         "linux fallback",
			xdgCacheHome: "",
			want:         filepath.Join(home, ".cache", "pagerduty-client"),
			skipNonLinux: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipNonDarwin && runtime.GOOS != "darwin" {
				t.Skip("darwin-only test")
			}
			if tt.skipNonLinux && runtime.GOOS != "linux" {
				t.Skip("linux-only test")
			}

			t.Setenv("XDG_CACHE_HOME", tt.xdgCacheHome)

			got, err := dirs.PdcCacheDir()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPdcConfigDir_AppendsPagerdutyClient(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/custom")

	got, err := dirs.PdcConfigDir()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/custom/pagerduty-client", got)
}

func TestPdcCacheDir_AppendsPagerdutyClient(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "/tmp/custom")

	got, err := dirs.PdcCacheDir()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/custom/pagerduty-client", got)
}
