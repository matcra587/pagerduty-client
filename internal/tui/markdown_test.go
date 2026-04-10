package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlamourStyle_DefaultIsDracula(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "dracula", glamourStyle("default"))
	assert.Equal(t, "dracula", glamourStyle("monokai"))
	assert.Equal(t, "dracula", glamourStyle(""))
}

func TestGlamourStyle_LightTheme(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "light", glamourStyle("catppuccin-latte"))
}

func TestGlamourStyle_Monochrome(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "notty", glamourStyle("monochrome"))
}

func TestGlamourStyle_DarkCatppuccin(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "dark", glamourStyle("catppuccin-frappe"))
	assert.Equal(t, "dark", glamourStyle("catppuccin-macchiato"))
	assert.Equal(t, "dark", glamourStyle("catppuccin-mocha"))
}

func TestRenderMarkdown_NonEmpty(t *testing.T) {
	t.Parallel()
	out := renderMarkdown("# Hello\n\nWorld", 80)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "World")
}

func TestRenderMarkdown_Empty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, renderMarkdown("", 80))
}
