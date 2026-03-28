package output

import (
	"bytes"
	"testing"

	"github.com/matcra587/pagerduty-client/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	incidents := testutil.LoadIncidents(t)
	err := RenderJSON(&buf, incidents, false)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"P000001"`)
	assert.Contains(t, buf.String(), `"High CPU on payment-api prod nodes"`)
}

func TestRenderJSON_Highlighted(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	data := map[string]string{"id": "P123", "status": "triggered"}
	err := RenderJSON(&buf, data, true)
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "P123")
	assert.Contains(t, out, "\x1b[") // ANSI escape sequences present
}

func TestRenderAgentJSON(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	incidents := testutil.LoadIncidents(t)
	err := RenderAgentJSON(&buf, "incident list", ResourceIncident, incidents, nil, nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"success":true`)
	assert.Contains(t, buf.String(), `"command":"incident list"`)
	assert.Contains(t, buf.String(), `"P000001"`)
}
