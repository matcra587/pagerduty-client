package table

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/gechr/clib/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTermWidth temporarily sets the package termWidth function
// for the duration of the test. Not parallel-safe.
func withTermWidth(t *testing.T, w int) {
	t.Helper()
	orig := termWidth
	termWidth = func() int { return w }
	t.Cleanup(func() { termWidth = orig })
}

func TestRender_PlainText(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, nil).
		AddCol(Col("ID")).
		AddCol(Col("Title")).
		Row("P1", "CPU High").
		Row("P2", "Disk Full").
		Render()
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "Title")
	assert.Contains(t, out, "P1")
	assert.Contains(t, out, "CPU High")
	assert.Contains(t, out, "P2")
	assert.Contains(t, out, "Disk Full")
}

func TestRender_EmptyTable(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, nil).Render()
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestRender_Sanitize(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, nil).
		AddCol(Col("Val")).
		Row("hello\x00world").
		Render()
	require.NoError(t, err)
	out := buf.String()
	assert.NotContains(t, out, "\x00")
	assert.Contains(t, out, "hello")
}

func TestRender_TimeAgo(t *testing.T) {
	t.Parallel()
	ts := time.Now().Add(-15 * time.Minute).UTC().Format(time.RFC3339)
	var buf bytes.Buffer
	err := New(&buf, nil).
		AddCol(Col("When").TimeAgo()).
		Row(ts).
		Render()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "ago")
}

func TestRender_TimeAgo_Invalid(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, nil).
		AddCol(Col("When").TimeAgo()).
		Row("not-a-timestamp").
		Render()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "not-a-timestamp")
}

func TestRender_ColumnAlignment(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, nil).
		AddCol(Col("Short")).
		AddCol(Col("Second")).
		Row("a", "x").
		Row("longer", "y").
		Render()
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, 3)
	headerIdx := strings.Index(lines[0], "Second")
	row1Idx := strings.Index(lines[1], "x")
	row2Idx := strings.Index(lines[2], "y")
	assert.Equal(t, headerIdx, row1Idx)
	assert.Equal(t, headerIdx, row2Idx)
}

func TestRender_Truncation(t *testing.T) {
	t.Parallel()
	longVal := strings.Repeat("A", 100)
	var buf bytes.Buffer
	err := New(&buf, nil).
		AddCol(Col("Val")).
		Row(longVal).
		Render()
	require.NoError(t, err)
	out := buf.String()
	assert.Contains(t, out, "AAA...")
	assert.NotContains(t, out, longVal)
}

func TestRender_ExcessRowValues(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, nil).
		AddCol(Col("A")).
		Row("one", "extra", "values").
		Render()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "one")
}

func TestRender_FewerRowValues(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, nil).
		AddCol(Col("A")).
		AddCol(Col("B")).
		Row("one").
		Render()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "one")
}

func TestRender_Themed_DimDefault(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, theme.Default()).
		AddCol(Col("Val")).
		Row("hello").
		Render()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "\x1b[")
}

func TestRender_Themed_BoldHeader(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, theme.Default()).
		AddCol(Col("Header")).
		Row("val").
		Render()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "\x1b[1m")
}

func TestRender_Themed_StyleMap(t *testing.T) {
	t.Parallel()
	m := map[string]lipgloss.Style{
		"hot": lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
	}
	var buf bytes.Buffer
	err := New(&buf, theme.Default()).
		AddCol(Col("Temp").StyleMap(m)).
		Row("hot").
		Render()
	require.NoError(t, err)
	// True-colour red: ESC[38;2;255;0;0m
	assert.Contains(t, buf.String(), "38;2;255;0;0")
}

func TestRender_Themed_StyleMap_Fallback(t *testing.T) {
	t.Parallel()
	m := map[string]lipgloss.Style{
		"hot": lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
	}
	var buf bytes.Buffer
	err := New(&buf, theme.Default()).
		AddCol(Col("Temp").StyleMap(m)).
		Row("cold").
		Render()
	require.NoError(t, err)
	out := buf.String()
	// Unmatched value should fall back to dim (some ANSI present)
	// but not have red colouring.
	assert.NotContains(t, out, "38;2;255;0;0")
	assert.Contains(t, out, "\x1b[")
}

func TestRender_Themed_Normal(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, theme.Default()).
		AddCol(Col("Plain").Normal()).
		AddCol(Col("Dim")).
		Row("alpha", "beta").
		Render()
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	dataLine := lines[1]

	// The data line must contain faint ANSI (from the dim column).
	assert.Contains(t, dataLine, "\x1b[2m", "Dim column should contain faint ANSI")

	// The Normal column value "alpha" must NOT be wrapped in faint.
	// In the rendered line, "alpha" appears before any ANSI for the
	// dim column. Verify the faint marker does not precede "alpha".
	alphaIdx := strings.Index(dataLine, "alpha")
	faintIdx := strings.Index(dataLine, "\x1b[2m")
	require.GreaterOrEqual(t, alphaIdx, 0)
	require.GreaterOrEqual(t, faintIdx, 0)
	assert.Greater(t, faintIdx, alphaIdx, "faint ANSI should appear after normal column text")
}

func TestRender_Themed_Bold(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, theme.Default()).
		AddCol(Col("Field").Bold()).
		Row("value").
		Render()
	require.NoError(t, err)
	// Data row should contain bold ANSI.
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	assert.Contains(t, lines[1], "\x1b[1m")
}

func TestRender_Themed_Link(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, theme.Default()).
		AddCol(Col("Name").Link(func(v string) string {
			return "https://example.com/" + v
		})).
		Row("foo").
		Render()
	require.NoError(t, err)
	out := buf.String()
	// OSC8 hyperlink: ESC]8;;URL ST
	assert.Contains(t, out, "https://example.com/foo")
	// Underline for colour mode.
	assert.Contains(t, out, "\x1b[4m")
}

func TestRender_Themed_StyleFn(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	err := New(&buf, theme.Default()).
		AddCol(Col("Colour").Style(func(string) lipgloss.Style {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
		})).
		Row("green").
		Render()
	require.NoError(t, err)
	// True-colour green: ESC[38;2;0;255;0m
	assert.Contains(t, buf.String(), "38;2;0;255;0")
}

func TestCol_BuilderMethods(t *testing.T) {
	t.Parallel()

	t.Run("Flex+Normal", func(t *testing.T) {
		t.Parallel()
		c := Col("X").Flex().Normal()
		assert.True(t, c.flex)
		assert.True(t, c.normal)
	})

	t.Run("Bold", func(t *testing.T) {
		t.Parallel()
		c := Col("Y").Bold()
		assert.True(t, c.bold)
	})

	t.Run("TimeAgo", func(t *testing.T) {
		t.Parallel()
		c := Col("Z").TimeAgo()
		assert.True(t, c.timeAgo)
	})
}

func TestRender_Flex_TruncatesOnTTY(t *testing.T) {
	withTermWidth(t, 80)
	th := theme.Default()
	var buf bytes.Buffer
	tbl := New(&buf, th)
	tbl.AddCol(Col("ID"))
	tbl.AddCol(Col("Title").Flex())
	long := strings.Repeat("x", 200)
	tbl.Row("P1", long)
	require.NoError(t, tbl.Render())

	out := xansi.Strip(buf.String())
	assert.Contains(t, out, "...")
	assert.NotContains(t, out, long)
}

func TestRender_Flex_PipedNoTruncation(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	tbl := New(&buf, nil) // nil theme = piped mode
	tbl.AddCol(Col("ID"))
	tbl.AddCol(Col("Title").Flex())
	long := strings.Repeat("x", 200)
	tbl.Row("P1", long)
	require.NoError(t, tbl.Render())
	assert.Contains(t, buf.String(), long)
}

func TestRender_Flex_UnboundedIgnoresTerminal(t *testing.T) {
	withTermWidth(t, 80)
	th := theme.Default()
	var buf bytes.Buffer
	tbl := New(&buf, th).Unbounded()
	tbl.AddCol(Col("ID"))
	tbl.AddCol(Col("Title").Flex())
	long := strings.Repeat("x", 200)
	tbl.Row("P1", long)
	require.NoError(t, tbl.Render())

	out := xansi.Strip(buf.String())
	assert.Contains(t, out, long)
}

func TestRender_Flex_MultiFlexDistributes(t *testing.T) {
	withTermWidth(t, 100)
	th := theme.Default()
	var buf bytes.Buffer
	tbl := New(&buf, th)
	tbl.AddCol(Col("ID"))
	tbl.AddCol(Col("A").Flex())
	tbl.AddCol(Col("B").Flex())
	longA := strings.Repeat("a", 200)
	longB := strings.Repeat("b", 200)
	tbl.Row("P1", longA, longB)
	require.NoError(t, tbl.Render())

	out := xansi.Strip(buf.String())
	// Both flex columns truncated (distributed).
	assert.Contains(t, out, "...")
	assert.NotContains(t, out, longA)
	assert.NotContains(t, out, longB)
}

func TestRender_Flex_NarrowTerminalClampsToMin(t *testing.T) {
	withTermWidth(t, 20) // smaller than fixed columns
	th := theme.Default()
	var buf bytes.Buffer
	tbl := New(&buf, th)
	tbl.AddCol(Col("ID"))
	tbl.AddCol(Col("Title").Flex())
	tbl.Row("P000001", strings.Repeat("x", 50))
	require.NoError(t, tbl.Render())

	// Flex gets clamped to minFlexW = 10, so content truncates.
	out := xansi.Strip(buf.String())
	assert.Contains(t, out, "...")
}

func TestRender_Flex_WithTimeAgo(t *testing.T) {
	withTermWidth(t, 80)
	th := theme.Default()
	var buf bytes.Buffer
	tbl := New(&buf, th)
	tbl.AddCol(Col("ID"))
	tbl.AddCol(Col("When").TimeAgo().Flex())
	tbl.Row("P1", "2020-01-01T00:00:00Z")
	require.NoError(t, tbl.Render())

	out := xansi.Strip(buf.String())
	assert.Contains(t, out, "ago")
	assert.NotContains(t, out, "2020-01-01")
}
