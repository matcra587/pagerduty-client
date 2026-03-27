package api

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	t.Parallel()
	err := &APIError{StatusCode: 404, Code: 2001, Message: "Not Found"}
	require.ErrorContains(t, err, "404")
	require.ErrorContains(t, err, "Not Found")
}

func TestAPIError_Is(t *testing.T) {
	t.Parallel()
	err := &APIError{StatusCode: 404, Code: 2001, Message: "Not Found"}
	wrapped := fmt.Errorf("something failed: %w", err)

	require.ErrorIs(t, wrapped, ErrNotFound)
	assert.NotErrorIs(t, wrapped, ErrRateLimited)
}

func TestAPIError_Sentinels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      *APIError
		sentinel error
		want     bool
	}{
		{"404 matches ErrNotFound", &APIError{StatusCode: 404}, ErrNotFound, true},
		{"429 matches ErrRateLimited", &APIError{StatusCode: 429}, ErrRateLimited, true},
		{"500 does not match ErrNotFound", &APIError{StatusCode: 500}, ErrNotFound, false},
		{"404 does not match ErrRateLimited", &APIError{StatusCode: 404}, ErrRateLimited, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.want {
				assert.ErrorIs(t, tt.err, tt.sentinel)
			} else {
				assert.NotErrorIs(t, tt.err, tt.sentinel)
			}
		})
	}
}
