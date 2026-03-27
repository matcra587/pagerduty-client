package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectFormat_AgentMode(t *testing.T) {
	t.Parallel()
	f := DetectFormat(FormatOpts{AgentMode: true})
	assert.Equal(t, FormatAgentJSON, f)
}

func TestDetectFormat_ExplicitJSON(t *testing.T) {
	t.Parallel()
	f := DetectFormat(FormatOpts{Format: "json"})
	assert.Equal(t, FormatJSON, f)
}

func TestDetectFormat_ExplicitTable(t *testing.T) {
	t.Parallel()
	f := DetectFormat(FormatOpts{Format: "table", IsTTY: true})
	assert.Equal(t, FormatTable, f)
}

func TestDetectFormat_NonTTY(t *testing.T) {
	t.Parallel()
	f := DetectFormat(FormatOpts{IsTTY: false})
	assert.Equal(t, FormatPlainTable, f)
}

func TestDetectFormat_TTYDefault(t *testing.T) {
	t.Parallel()
	f := DetectFormat(FormatOpts{IsTTY: true})
	assert.Equal(t, FormatTable, f)
}
