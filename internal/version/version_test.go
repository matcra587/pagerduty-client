package version_test

import (
	"strings"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfo(t *testing.T) {
	t.Parallel()
	info := version.Info()

	require.NotEmpty(t, info.Version)
	assert.Equal(t, "dev", info.Version)
	assert.Equal(t, "unknown", info.Commit)
	assert.Equal(t, "unknown", info.Branch)
	assert.Equal(t, "unknown", info.BuildTime)
	assert.Equal(t, "unknown", info.BuildBy)
}

func TestBuildInfo_String(t *testing.T) {
	t.Parallel()
	info := version.Info()
	s := info.String()

	assert.True(t, strings.HasPrefix(s, "pdc "), "expected prefix 'pdc ', got: %q", s)
	assert.Contains(t, s, info.Version)
	assert.Contains(t, s, info.Commit)
	assert.Contains(t, s, info.Branch)
	assert.Contains(t, s, info.BuildTime)
	assert.Contains(t, s, info.BuildBy)
}

func TestResolvedCommit_LdflagsSet(t *testing.T) {
	t.Parallel()
	// In test binaries, Commit is "unknown" and vcs.revision is
	// available from debug.ReadBuildInfo. ResolvedCommit should
	// return a non-empty string in either case.
	got := version.ResolvedCommit()
	assert.NotEmpty(t, got)
}
