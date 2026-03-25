package tui

import (
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveFieldPath(t *testing.T) {
	tests := []struct {
		name   string
		body   map[string]any
		path   string
		want   any
		wantOK bool
	}{
		{
			name:   "top-level key",
			body:   map[string]any{"status": "triggered"},
			path:   "status",
			want:   "triggered",
			wantOK: true,
		},
		{
			name:   "nested path",
			body:   map[string]any{"details": map[string]any{"body": "hello"}},
			path:   "details.body",
			want:   "hello",
			wantOK: true,
		},
		{
			name:   "deeply nested path",
			body:   map[string]any{"a": map[string]any{"b": map[string]any{"c": 42}}},
			path:   "a.b.c",
			want:   42,
			wantOK: true,
		},
		{
			name:   "missing key returns false",
			body:   map[string]any{"status": "triggered"},
			path:   "missing",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "missing nested key returns false",
			body:   map[string]any{"details": map[string]any{"body": "hello"}},
			path:   "details.missing",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "non-map mid-path returns false",
			body:   map[string]any{"details": "not a map"},
			path:   "details.body",
			want:   nil,
			wantOK: false,
		},
		{
			name: "cef_details fallback",
			body: map[string]any{
				"cef_details": map[string]any{
					"details": map[string]any{"body": "from cef"},
				},
			},
			path:   "details.body",
			want:   "from cef",
			wantOK: true,
		},
		{
			name: "direct path preferred over cef_details",
			body: map[string]any{
				"details": map[string]any{"body": "direct"},
				"cef_details": map[string]any{
					"details": map[string]any{"body": "from cef"},
				},
			},
			path:   "details.body",
			want:   "direct",
			wantOK: true,
		},
		{
			name:   "empty body returns false",
			body:   map[string]any{},
			path:   "anything",
			want:   nil,
			wantOK: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveFieldPath(tt.body, tt.path)
			require.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBodySection_UsesNormaliser_GCP(t *testing.T) {
	m := incidentDetail{
		incident: testIncidents()[0],
		alerts: []pagerduty.IncidentAlert{{
			Body: map[string]any{
				"details": map[string]any{
					"custom_details": map[string]any{
						"incident": map[string]any{
							"policy_name":     "Uptime failure",
							"condition_name":  "Check failed",
							"condition":       map[string]any{},
							"observed_value":  "2.0",
							"threshold_value": "1",
							"state":           "open",
							"url":             "https://console.cloud.google.com/test",
						},
						"version": "1.2",
					},
				},
			},
		}},
		width: 80,
	}
	body := m.bodySection()
	assert.Contains(t, body, "Google Cloud Monitoring")
	assert.Contains(t, body, "Uptime failure")
	assert.Contains(t, body, "Check failed")
	assert.NotContains(t, body, "map[")
}
