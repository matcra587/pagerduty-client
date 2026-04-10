package theme_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	clibtheme "github.com/gechr/clib/theme"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_DefaultTheme(t *testing.T) {
	th := theme.Resolve("")
	require.NotNil(t, th)
	assert.NotNil(t, th.Red)
	assert.NotNil(t, th.Green)
}

func TestResolve_NamedPreset(t *testing.T) {
	th := theme.Resolve("dracula")
	require.NotNil(t, th)
	assert.NotNil(t, th.Red)
}

func TestResolve_UnknownFallsBack(t *testing.T) {
	th := theme.Resolve("nonexistent")
	require.NotNil(t, th, "unknown preset should fall back to default")
}

func TestPresetNames_ContainsDracula(t *testing.T) {
	names := theme.PresetNames()
	assert.Contains(t, names, "dracula")
	assert.Contains(t, names, "monochrome")
}

func TestResolve_EnvVarOverridesEmptyName(t *testing.T) {
	t.Setenv("PDC_THEME", "monokai")
	clibtheme.SetEnvPrefix("PDC")
	t.Cleanup(func() { clibtheme.SetEnvPrefix("") })

	th := theme.Resolve("")
	require.NotNil(t, th)
	assert.Equal(t, "monokai", th.String())
}

func TestApply_UpdatesDerivedStyles(t *testing.T) {
	theme.ResetForTest()
	light := theme.Resolve("catppuccin-latte")
	theme.Apply(light)
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	assert.Equal(t, light, theme.Theme, "Theme should be set to light preset")
	assert.Equal(t, light.Red.GetForeground(), theme.UrgencyHigh.GetForeground(),
		"UrgencyHigh foreground should match light theme red")

	for _, p := range []string{"P1", "P2", "P3", "P4", "P5"} {
		_, ok := theme.PriorityStyle(p)
		assert.True(t, ok, "PriorityStyles missing %s after Apply", p)
	}
}

func TestApply_ChromeDerivedFromTheme(t *testing.T) {
	theme.ResetForTest()
	th := theme.Resolve("dracula")
	theme.Apply(th)
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	assert.Equal(t, th.Blue.GetForeground(), theme.ColorHeaderFg,
		"header fg should match theme blue")
	assert.Equal(t, th.MarkdownText.GetForeground(), theme.ColorStatusBarFg,
		"status bar fg should match theme markdown text")
}

func TestApply_SecondCallIsNoop(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("catppuccin-latte"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	lightRed := theme.UrgencyHigh.GetForeground()
	theme.Apply(theme.Resolve("default"))
	assert.Equal(t, lightRed, theme.UrgencyHigh.GetForeground(),
		"second Apply should be a no-op")
}

func TestEntityColor_Deterministic(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	s1 := theme.EntityColor("web-api")
	s2 := theme.EntityColor("web-api")
	assert.Equal(t, s1.GetForeground(), s2.GetForeground(),
		"same name should always produce the same colour")
}

func TestEntityColor_EmptyReturnsPlain(t *testing.T) {
	s := theme.EntityColor("")
	plain := lipgloss.NewStyle()
	assert.Equal(t, plain.GetForeground(), s.GetForeground(),
		"empty name should return unstyled")
}

func TestEntityColor_DifferentNamesVary(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	// With 20 palette colours, two short distinct strings are very likely
	// to hash to different indices.
	s1 := theme.EntityColor("service-alpha")
	s2 := theme.EntityColor("service-zeta")
	assert.NotEqual(t, s1.GetForeground(), s2.GetForeground(),
		"different names should usually produce different colours")
}

func TestPriorityStyle_UnknownReturnsFalse(t *testing.T) {
	_, ok := theme.PriorityStyle("P9")
	assert.False(t, ok, "unknown priority should return false")
}

func TestRenderEntityNames_Nil(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	assert.Empty(t, theme.RenderEntityNames(nil))
}

func TestRenderEntityNames_Empty(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	assert.Empty(t, theme.RenderEntityNames([]string{}))
}

func TestRenderEntityNames_Single(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	result := theme.RenderEntityNames([]string{"Alice"})
	assert.Contains(t, result, "Alice")
	// Should have ANSI colour codes wrapping the name.
	assert.NotEqual(t, "Alice", result, "name should be styled with colour")
}

func TestRenderEntityNames_Multiple(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	result := theme.RenderEntityNames([]string{"Alice", "Bob"})
	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "Bob")
	assert.Contains(t, result, ", ", "names should be joined with comma separator")
}

func TestRenderEntityNames_PositionIndependence(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	ab := theme.RenderEntityNames([]string{"Alice", "Bob"})
	ba := theme.RenderEntityNames([]string{"Bob", "Alice"})

	// Extract the rendered "Alice" segment from each result.
	// The separator ", " is uncoloured so we can split on it.
	abParts := strings.SplitN(ab, ", ", 2)
	baParts := strings.SplitN(ba, ", ", 2)

	require.Len(t, abParts, 2)
	require.Len(t, baParts, 2)

	// Alice is first in ab, second in ba.
	assert.Equal(t, abParts[0], baParts[1],
		"Alice should have the same colour regardless of position")
}

func TestRenderEntityNames_Duplicates(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	result := theme.RenderEntityNames([]string{"Alice", "Alice"})
	parts := strings.SplitN(result, ", ", 2)
	require.Len(t, parts, 2)
	assert.Equal(t, parts[0], parts[1],
		"duplicate names should produce identical styled output")
}

func TestRenderEntityNames_SkipsEmpty(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Resolve("default"))
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Resolve("default"))
	})

	result := theme.RenderEntityNames([]string{"Alice", "", "  ", "Bob"})
	// Should only contain Alice and Bob, no empty segments.
	parts := strings.SplitN(result, ", ", 3)
	assert.Len(t, parts, 2, "empty and whitespace-only elements should be skipped")
}
