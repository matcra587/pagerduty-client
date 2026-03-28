package update_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		goos     string
		goarch   string
		version  string
		expected string
	}{
		{
			name:     "linux amd64",
			goos:     "linux",
			goarch:   "amd64",
			version:  "0.4.0",
			expected: "pagerduty-client_0.4.0_linux_amd64.tar.gz",
		},
		{
			name:     "darwin arm64",
			goos:     "darwin",
			goarch:   "arm64",
			version:  "0.4.0",
			expected: "pagerduty-client_0.4.0_darwin_arm64.tar.gz",
		},
		{
			name:     "windows amd64",
			goos:     "windows",
			goarch:   "amd64",
			version:  "0.4.0",
			expected: "pagerduty-client_0.4.0_windows_amd64.zip",
		},
		{
			name:     "linux arm64",
			goos:     "linux",
			goarch:   "arm64",
			version:  "1.2.3",
			expected: "pagerduty-client_1.2.3_linux_arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := update.AssetName(tt.version, tt.goos, tt.goarch)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	t.Parallel()

	data := []byte("fake binary content")
	hash := sha256.Sum256(data)
	checksumLine := fmt.Sprintf("%x  pdc_test.tar.gz", hash)

	t.Run("valid checksum", func(t *testing.T) {
		t.Parallel()
		assert.True(t, update.VerifyChecksum(data, checksumLine, "pdc_test.tar.gz"))
	})

	t.Run("wrong hash", func(t *testing.T) {
		t.Parallel()
		badLine := "0000000000000000000000000000000000000000000000000000000000000000  pdc_test.tar.gz"
		assert.False(t, update.VerifyChecksum(data, badLine, "pdc_test.tar.gz"))
	})

	t.Run("wrong asset name", func(t *testing.T) {
		t.Parallel()
		assert.False(t, update.VerifyChecksum(data, checksumLine, "wrong_name.tar.gz"))
	})

	t.Run("malformed line", func(t *testing.T) {
		t.Parallel()
		assert.False(t, update.VerifyChecksum(data, "nospaces", "pdc_test.tar.gz"))
	})
}

func TestAtomicReplace(t *testing.T) {
	t.Parallel()

	t.Run("replaces existing file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "pdc")
		require.NoError(t, os.WriteFile(target, []byte("old"), 0o600))

		err := update.AtomicReplace(target, []byte("new"))
		require.NoError(t, err)

		got, err := os.ReadFile(target) //nolint:gosec // test reads from t.TempDir()
		require.NoError(t, err)
		assert.Equal(t, "new", string(got))

		info, err := os.Stat(target)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	})

	t.Run("creates file if missing", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		target := filepath.Join(dir, "pdc")

		err := update.AtomicReplace(target, []byte("brand new"))
		require.NoError(t, err)

		got, err := os.ReadFile(target) //nolint:gosec // test reads from t.TempDir()
		require.NoError(t, err)
		assert.Equal(t, "brand new", string(got))
	})

	t.Run("fails on invalid directory", func(t *testing.T) {
		t.Parallel()

		target := filepath.Join(t.TempDir(), "nonexistent", "subdir", "pdc")
		err := update.AtomicReplace(target, []byte("data"))
		assert.Error(t, err)
	})
}
