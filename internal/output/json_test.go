package output

import (
	"bytes"
	"testing"

	"github.com/gechr/clib/theme"
	"github.com/matcra587/pagerduty-client/internal/compact"
	"github.com/matcra587/pagerduty-client/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	incidents := testutil.LoadIncidents(t)
	err := RenderJSON(&buf, incidents, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"P000001"`)
	assert.Contains(t, buf.String(), `"High CPU on payment-api prod nodes"`)
}

func TestRenderJSON_Highlighted(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	data := map[string]string{"id": "P123", "status": "triggered"}
	th := theme.Default()
	err := RenderJSON(&buf, data, th)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "P123")
	assert.Contains(t, out, "\x1b[") // ANSI escape sequences present
}

func TestRenderAgentJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	incidents := testutil.LoadIncidents(t)
	err := RenderAgentJSON(&buf, "incident list", compact.ResourceIncident, incidents, nil, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"success":true`)
	assert.Contains(t, buf.String(), `"command":"incident list"`)
	assert.Contains(t, buf.String(), `"P000001"`)
}

func TestChromaStyle_DefaultIsMonokai(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "monokai", chromaStyle("default"))
	assert.Equal(t, "monokai", chromaStyle("monokai"))
	assert.Equal(t, "monokai", chromaStyle(""))
}

func TestChromaStyle_Dracula(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "dracula", chromaStyle("dracula"))
}

func TestChromaStyle_Catppuccin(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "catppuccin-latte", chromaStyle("catppuccin-latte"))
	assert.Equal(t, "catppuccin-frappe", chromaStyle("catppuccin-frappe"))
	assert.Equal(t, "catppuccin-macchiato", chromaStyle("catppuccin-macchiato"))
	assert.Equal(t, "catppuccin-mocha", chromaStyle("catppuccin-mocha"))
}

func TestChromaStyle_Monochrome(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "bw", chromaStyle("monochrome"))
}
