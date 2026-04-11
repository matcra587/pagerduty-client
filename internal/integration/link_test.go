package integration

import (
	"testing"

	"github.com/matcra587/pagerduty-client/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveExternalLink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *config.Config
		body map[string]any
		want string
	}{
		{
			name: "custom field link",
			cfg: &config.Config{
				CustomFields: []config.CustomField{
					{Path: "dashboard_url", Display: "link"},
				},
			},
			body: map[string]any{
				"dashboard_url": "https://grafana.example.com/d/abc",
			},
			want: "https://grafana.example.com/d/abc",
		},
		{
			name: "integration detected link",
			cfg:  nil,
			body: map[string]any{
				"cef_details": map[string]any{
					"client":     "Datadog",
					"client_url": "https://app.datadoghq.com/monitors/123",
					"details": map[string]any{
						"query": "avg:cpu > 90",
					},
				},
			},
			want: "https://app.datadoghq.com/monitors/123",
		},
		{
			name: "no link found",
			cfg:  nil,
			body: map[string]any{"foo": "bar"},
			want: "",
		},
		{
			name: "nil body",
			cfg:  nil,
			body: nil,
			want: "",
		},
		{
			name: "non-http rejected",
			cfg: &config.Config{
				CustomFields: []config.CustomField{
					{Path: "evil", Display: "link"},
				},
			},
			body: map[string]any{
				"evil": "javascript:alert(1)",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ResolveExternalLink(tt.cfg, tt.body)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveFieldPath(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			got, ok := ResolveFieldPath(tt.body, tt.path)
			require.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
