package update

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunHomebrew_StableRefreshesTapBeforeUpgrade(t *testing.T) {
	testRunHomebrew(t, ChannelStable, []string{
		"brew --repo matcra587/tap",
		"git -C TAP_REPO pull --ff-only",
		"brew upgrade matcra587/tap/pagerduty-client",
	})
}

func testRunHomebrew(t *testing.T, channel Channel, want []string) {
	t.Helper()

	logPath := filepath.Join(t.TempDir(), "brew.log")
	brewDir := t.TempDir()
	brewPath := filepath.Join(brewDir, "brew")
	tapRepo := t.TempDir()
	gitPath := filepath.Join(brewDir, "git")

	script := "#!/bin/sh\nprintf 'brew %s\\n' \"$*\" >> \"$PDC_TEST_BREW_LOG\"\nif [ \"$1\" = \"--repo\" ]; then\n  printf '%s\\n' \"$PDC_TEST_TAP_REPO\"\nfi\n"
	require.NoError(t, os.WriteFile(brewPath, []byte(script), 0o755))                                                            //nolint:gosec // stub script must be executable
	require.NoError(t, os.WriteFile(gitPath, []byte("#!/bin/sh\nprintf 'git %s\\n' \"$*\" >> \"$PDC_TEST_BREW_LOG\"\n"), 0o755)) //nolint:gosec // stub script must be executable

	t.Setenv("PATH", brewDir)
	t.Setenv("PDC_TEST_BREW_LOG", logPath)
	t.Setenv("PDC_TEST_TAP_REPO", tapRepo)

	require.NoError(t, runHomebrew(context.Background(), channel))

	data, err := os.ReadFile(logPath) //nolint:gosec // logPath is rooted in t.TempDir()
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for i := range want {
		want[i] = strings.ReplaceAll(want[i], "TAP_REPO", tapRepo)
	}
	require.Equal(t, want, lines)
}
