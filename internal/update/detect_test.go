package update_test

import (
	"testing"

	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/stretchr/testify/assert"
)

func TestDetectMethodFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		bin    string
		goMod  string
		expect update.InstallMethod
	}{
		{"homebrew macOS arm", "/opt/homebrew/Cellar/pagerduty-client/0.3.1/bin/pdc", "", update.Homebrew},
		{"homebrew macOS intel", "/usr/local/Cellar/pagerduty-client/0.3.1/bin/pdc", "", update.Homebrew},
		{"homebrew linux", "/home/linuxbrew/.linuxbrew/Cellar/pagerduty-client/0.3.1/bin/pdc", "", update.Homebrew},
		{"go install with module", "/home/user/go/bin/pdc", "github.com/matcra587/pagerduty-client", update.GoInstall},
		{"go install gobin", "/custom/gobin/pdc", "github.com/matcra587/pagerduty-client", update.GoInstall},
		{"binary fallback", "/usr/local/bin/pdc", "", update.Binary},
		{"binary in home", "/home/user/.local/bin/pdc", "", update.Binary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := update.DetectMethodFromPath(tt.bin, tt.goMod)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestInstallMethod_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method update.InstallMethod
		expect string
	}{
		{update.Binary, "binary"},
		{update.Homebrew, "homebrew"},
		{update.GoInstall, "go install"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expect, tt.method.String())
		})
	}
}
