package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFromEmail(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"user@example.com", "user@example.com", false},
		{"user@example", "user@example", false}, // valid per RFC 5322
		{"user.example.com", "", true},          // no @
		{"", "", true},
		{"@.", "", true},
		{"a@.", "", true},
		{"user@", "", true},
		{"@example.com", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseFromEmail(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
