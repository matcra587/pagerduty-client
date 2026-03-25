package components

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// PersistBg
// ---------------------------------------------------------------------------

func TestPersistBg_EmptyBg_ReturnsLineUnchanged(t *testing.T) {
	line := "hello world"
	assert.Equal(t, line, PersistBg(line, ""))
}

func TestPersistBg_EmptyLine_ReturnsBg(t *testing.T) {
	bg := "\x1b[48;2;10;20;30m"
	assert.Equal(t, bg, PersistBg("", bg))
}

func TestPersistBg_NoANSI_PrependsBg(t *testing.T) {
	bg := "\x1b[48;2;1;2;3m"
	got := PersistBg("plain text", bg)
	assert.Equal(t, bg+"plain text", got)
}

func TestPersistBg_SingleSGR_ReappliesBgAfterIt(t *testing.T) {
	bg := "\x1b[48;2;0;0;0m"
	sgr := "\x1b[1m" // bold
	line := sgr + "bold"
	got := PersistBg(line, bg)
	assert.Equal(t, bg+sgr+bg+"bold", got)
}

func TestPersistBg_MultipleSGR_ReappliesBgAfterEach(t *testing.T) {
	bg := "\x1b[48;2;5;5;5m"
	sgr1 := "\x1b[1m"  // bold
	sgr2 := "\x1b[31m" // red fg
	line := sgr1 + "a" + sgr2 + "b"
	got := PersistBg(line, bg)
	assert.Equal(t, bg+sgr1+bg+"a"+sgr2+bg+"b", got)
}

func TestPersistBg_NonSGREscape_NotMatched(t *testing.T) {
	bg := "\x1b[48;2;0;0;0m"
	cursor := "\x1b[2A" // cursor-up, ends with 'A' not 'm'
	line := cursor + "text"
	got := PersistBg(line, bg)
	// The cursor-move escape should pass through byte-by-byte, bg only prepended.
	assert.Contains(t, got, "text")
	// Should not inject an extra bg after the cursor escape.
	assert.Equal(t, 1, strings.Count(got, bg), "bg should appear only once (the prepend)")
}

func TestPersistBg_MalformedCSI_NotMatched(t *testing.T) {
	bg := "\x1b[48;2;0;0;0m"
	malformed := "\x1b[12" // no closing 'm'
	line := malformed + "rest"
	got := PersistBg(line, bg)
	assert.Equal(t, 1, strings.Count(got, bg), "bg should appear only once (the prepend)")
	assert.Contains(t, got, "rest")
}

func TestPersistBg_MultiByte(t *testing.T) {
	// Japanese text with an SGR sequence in the middle.
	line := "こんにちは\x1b[31m世界\x1b[0m"
	bg := "\x1b[48;2;34;34;51m"
	result := PersistBg(line, bg)
	// Must contain the original text intact (not corrupted).
	assert.Contains(t, result, "こんにちは")
	assert.Contains(t, result, "世界")
}

// ---------------------------------------------------------------------------
// ColorToANSIBg
// ---------------------------------------------------------------------------

func TestColorToANSIBg_Black(t *testing.T) {
	c := color.RGBA{R: 0, G: 0, B: 0, A: 0xFF}
	assert.Equal(t, "\x1b[48;2;0;0;0m", ColorToANSIBg(c))
}

func TestColorToANSIBg_White(t *testing.T) {
	c := color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	assert.Equal(t, "\x1b[48;2;255;255;255m", ColorToANSIBg(c))
}

func TestColorToANSIBg_SpecificColour(t *testing.T) {
	c := color.RGBA{R: 0x1A, G: 0x1A, B: 0x2E, A: 0xFF}
	assert.Equal(t, "\x1b[48;2;26;26;46m", ColorToANSIBg(c))
}

// ---------------------------------------------------------------------------
// RenderOverlay
// ---------------------------------------------------------------------------

func TestRenderOverlay_ContainsContent(t *testing.T) {
	out := RenderOverlay("hello\nworld", 0)
	assert.Contains(t, out, "hello", "output should contain first content line")
	assert.Contains(t, out, "world", "output should contain second content line")
}

func TestRenderOverlay_MinWidthRespected(t *testing.T) {
	minW := 60
	out := RenderOverlay("short", minW)
	// lipgloss.Width accounts for ANSI escapes, giving visible width.
	require.GreaterOrEqual(t, lipgloss.Width(out), minW)
}

func TestRenderOverlay_MultiLineBgApplied(t *testing.T) {
	content := "line one\nline two\nline three"
	out := RenderOverlay(content, 0)

	bg := ColorToANSIBg(theme.ColorOverlayBg)
	// Content lines (not border lines) must carry the overlay bg escape.
	var matched int
	for line := range strings.SplitSeq(out, "\n") {
		if strings.Contains(line, bg) {
			matched++
		}
	}
	assert.GreaterOrEqual(t, matched, 3, "at least 3 lines should contain overlay bg escape")
}
