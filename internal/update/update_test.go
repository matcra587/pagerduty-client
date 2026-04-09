package update_test

import (
	"testing"

	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/stretchr/testify/assert"
)

func TestChannelMismatchErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		bin     string
		goMod   string
		channel update.Channel
		wantErr string
	}{
		{
			name:    "homebrew stable install with dev channel",
			bin:     "/opt/homebrew/Cellar/pagerduty-client/0.7.0/bin/pdc",
			goMod:   "",
			channel: update.ChannelDev,
			wantErr: "brew uninstall pagerduty-client && brew install --HEAD",
		},
		{
			name:    "homebrew HEAD install with stable channel",
			bin:     "/opt/homebrew/Cellar/pagerduty-client/HEAD-abc1234/bin/pdc",
			goMod:   "",
			channel: update.ChannelStable,
			wantErr: "brew uninstall pagerduty-client && brew install matcra587",
		},
		{
			name:    "binary with dev channel",
			bin:     "/usr/local/bin/pdc",
			goMod:   "",
			channel: update.ChannelDev,
			wantErr: "dev channel is not supported for standalone binaries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := update.ValidateMethodChannel(
				update.DetectMethodFromPath(tt.bin, tt.goMod),
				update.IsHomebrewHEADFromPath(tt.bin),
				tt.channel,
			)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestValidateMethodChannel_ValidCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		method  update.InstallMethod
		isHEAD  bool
		channel update.Channel
	}{
		{"homebrew stable with stable", update.Homebrew, false, update.ChannelStable},
		{"homebrew HEAD with dev", update.Homebrew, true, update.ChannelDev},
		{"go install with stable", update.GoInstall, false, update.ChannelStable},
		{"go install with dev", update.GoInstall, false, update.ChannelDev},
		{"binary with stable", update.Binary, false, update.ChannelStable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := update.ValidateMethodChannel(tt.method, tt.isHEAD, tt.channel)
			assert.NoError(t, err)
		})
	}
}
