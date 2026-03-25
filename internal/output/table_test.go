package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTable(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"ID", "Title", "Status"}
	rows := [][]string{
		{"P1", "CPU High", "triggered"},
		{"P2", "Disk Full", "acknowledged"},
	}
	err := RenderTable(&buf, headers, rows, false)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "P1")
	assert.Contains(t, output, "CPU High")
	assert.Contains(t, output, "ID")
}
