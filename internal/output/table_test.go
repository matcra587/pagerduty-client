package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gechr/clib/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTable(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	headers := []string{"ID", "Title", "Status"}
	rows := [][]string{
		{"P1", "CPU High", "triggered"},
		{"P2", "Disk Full", "acknowledged"},
	}
	err := RenderTable(&buf, headers, rows, nil)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "P1")
	assert.Contains(t, output, "CPU High")
	assert.Contains(t, output, "ID")
}

func TestRenderTable_Truncation(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	headers := []string{"ID", "Title"}
	longTitle := strings.Repeat("A", 100)
	rows := [][]string{
		{"P1", longTitle},
	}
	err := RenderTable(&buf, headers, rows, nil)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "AAA...")
	assert.NotContains(t, output, longTitle)
}

func TestRenderTable_SanitisesControlChars(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	headers := []string{"ID", "Title"}
	rows := [][]string{
		{"P1", "Normal \x1b[2Jpwned"},
	}
	err := RenderTable(&buf, headers, rows, nil)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "^[")
	assert.NotContains(t, output, "\x1b")
}

func TestRenderTable_Colour(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	headers := []string{"ID", "Status", "Urgency"}
	rows := [][]string{
		{"P1", "triggered", "high"},
	}
	err := RenderTable(&buf, headers, rows, theme.Default())
	require.NoError(t, err)
	output := buf.String()
	// Colour output contains ANSI escape codes.
	assert.Contains(t, output, "\x1b[")
	assert.Contains(t, output, "triggered")
	assert.Contains(t, output, "high")
}

func TestRenderTable_ThemeColours(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	headers := []string{"ID", "Status", "Urgency"}
	rows := [][]string{
		{"P1", "triggered", "high"},
	}
	th := theme.Default()
	err := RenderTable(&buf, headers, rows, th)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "\x1b[")
	assert.Contains(t, output, "triggered")
}
