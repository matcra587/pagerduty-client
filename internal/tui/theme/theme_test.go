package theme_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/matcra587/pagerduty-client/internal/tui/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresets_AllKeysValid(t *testing.T) {
	for name, ctor := range theme.Presets {
		t.Run(name, func(t *testing.T) {
			th := ctor()
			require.NotNil(t, th, "preset %q returned nil theme", name)
			assert.NotNil(t, th.Red, "preset %q missing Red style", name)
			assert.NotNil(t, th.Green, "preset %q missing Green style", name)
			assert.NotNil(t, th.Yellow, "preset %q missing Yellow style", name)
			assert.NotNil(t, th.Blue, "preset %q missing Blue style", name)
		})
	}
}

func TestPresets_ContainsExpectedNames(t *testing.T) {
	for _, name := range []string{"dark", "light", "high-contrast"} {
		_, ok := theme.Presets[name]
		assert.True(t, ok, "missing preset %q", name)
	}
}

func TestApply_UpdatesDerivedStyles(t *testing.T) {
	theme.ResetForTest()
	light := theme.Presets["light"]()
	theme.Apply(light)
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	assert.Equal(t, light, theme.Theme, "Theme should be set to light preset")
	assert.Equal(t, light.Red.GetForeground(), theme.UrgencyHigh.GetForeground(),
		"UrgencyHigh foreground should match light theme red")

	for _, p := range []string{"P1", "P2", "P3", "P4", "P5"} {
		_, ok := theme.PriorityStyle(p)
		assert.True(t, ok, "PriorityStyles missing %s after Apply", p)
	}
}

func TestApply_LightChromeColors(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["light"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	r, g, b, _ := theme.ColorOverlayBg.RGBA()
	luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535
	assert.Greater(t, luminance, 0.5, "light theme overlay background should be bright")
}

func TestApply_DarkChromeColors(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	r, g, b, _ := theme.ColorOverlayBg.RGBA()
	luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535
	assert.Less(t, luminance, 0.5, "dark theme overlay background should be dark")
}

func TestApply_SecondCallIsNoop(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["light"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	lightRed := theme.UrgencyHigh.GetForeground()
	theme.Apply(theme.Presets["dark"]())
	assert.Equal(t, lightRed, theme.UrgencyHigh.GetForeground(),
		"second Apply should be a no-op")
}

func TestEntityColor_Deterministic(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
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
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
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
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	assert.Empty(t, theme.RenderEntityNames(nil))
}

func TestRenderEntityNames_Empty(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	assert.Empty(t, theme.RenderEntityNames([]string{}))
}

func TestRenderEntityNames_Single(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	result := theme.RenderEntityNames([]string{"Alice"})
	assert.Contains(t, result, "Alice")
	// Should have ANSI colour codes wrapping the name.
	assert.NotEqual(t, "Alice", result, "name should be styled with colour")
}

func TestRenderEntityNames_Multiple(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	result := theme.RenderEntityNames([]string{"Alice", "Bob"})
	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "Bob")
	assert.Contains(t, result, ", ", "names should be joined with comma separator")
}

func TestRenderEntityNames_PositionIndependence(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
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
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	result := theme.RenderEntityNames([]string{"Alice", "Alice"})
	parts := strings.SplitN(result, ", ", 2)
	require.Len(t, parts, 2)
	assert.Equal(t, parts[0], parts[1],
		"duplicate names should produce identical styled output")
}

func TestRenderEntityNames_SkipsEmpty(t *testing.T) {
	theme.ResetForTest()
	theme.Apply(theme.Presets["dark"]())
	t.Cleanup(func() {
		theme.ResetForTest()
		theme.Apply(theme.Presets["dark"]())
	})

	result := theme.RenderEntityNames([]string{"Alice", "", "  ", "Bob"})
	// Should only contain Alice and Bob, no empty segments.
	parts := strings.SplitN(result, ", ", 3)
	assert.Len(t, parts, 2, "empty and whitespace-only elements should be skipped")
}
