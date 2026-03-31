package output

import (
	"bytes"
	"testing"

	"github.com/gechr/clib/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderColumns_Empty(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := RenderColumns(&buf, nil, 80, nil)
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestRenderColumns_SingleColumn(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	// Terminal too narrow for two columns.
	err := RenderColumns(&buf, []string{"Alpha", "Bravo", "Charlie"}, 10, nil)
	require.NoError(t, err)
	assert.Equal(t, "Alpha\nBravo\nCharlie\n", buf.String())
}

func TestRenderColumns_MultiColumn(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	// 5 items, terminal width 40, items are 5 chars + 2 gap = 7 per col = 5 cols.
	// But let's use a controlled width: items are 10 chars, width 30 -> 2 cols.
	items := []string{"AAAAAAAAAA", "BBBBBBBBBB", "CCCCCCCCCC", "DDDDDDDDDD", "EEEEEEEEEE"}
	err := RenderColumns(&buf, items, 30, nil)
	require.NoError(t, err)

	// 2 columns, 3 rows, column-major order: A,B,C down col 1; D,E down col 2.
	lines := []string{
		"AAAAAAAAAA  DDDDDDDDDD",
		"BBBBBBBBBB  EEEEEEEEEE",
		"CCCCCCCCCC",
	}
	expected := lines[0] + "\n" + lines[1] + "\n" + lines[2] + "\n"
	assert.Equal(t, expected, buf.String())
}

func TestRenderColumns_ZeroWidth(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := RenderColumns(&buf, []string{"Hello"}, 0, nil)
	require.NoError(t, err)
	assert.Equal(t, "Hello\n", buf.String())
}

func TestRenderColumns_WithTheme(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	th := theme.Default()
	err := RenderColumns(&buf, []string{"Test"}, 80, th)
	require.NoError(t, err)
	// Themed output contains ANSI escape sequences.
	assert.Contains(t, buf.String(), "Test")
	assert.NotEqual(t, "Test\n", buf.String())
}
