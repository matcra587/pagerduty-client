package update_test

import (
	"testing"

	"github.com/matcra587/pagerduty-client/internal/update"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    update.Channel
		wantErr bool
	}{
		{"stable", "stable", update.ChannelStable, false},
		{"dev", "dev", update.ChannelDev, false},
		{"empty defaults to stable", "", update.ChannelStable, false},
		{"invalid", "nightly", update.ChannelUnknown, true},
		{"case sensitive", "Stable", update.ChannelUnknown, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := update.ParseChannel(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChannel_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		channel update.Channel
		want    string
	}{
		{update.ChannelUnknown, "unknown"},
		{update.ChannelStable, "stable"},
		{update.ChannelDev, "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.channel.String())
		})
	}
}
