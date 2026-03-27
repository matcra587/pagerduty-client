package theme_test

import (
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
	light := theme.Presets["light"]()
	theme.Apply(light)

	assert.Equal(t, light, theme.Theme, "Theme should be set to light preset")
	assert.Equal(t, light.Red.GetForeground(), theme.UrgencyHigh.GetForeground(),
		"UrgencyHigh foreground should match light theme red")

	for _, p := range []string{"P1", "P2", "P3", "P4", "P5"} {
		_, ok := theme.PriorityStyle(p)
		assert.True(t, ok, "PriorityStyles missing %s after Apply", p)
	}

	// Restore default to avoid polluting other tests.
	theme.Apply(theme.Presets["dark"]())
}

func TestApply_LightChromeColors(t *testing.T) {
	theme.Apply(theme.Presets["light"]())

	r, g, b, _ := theme.ColorOverlayBg.RGBA()
	luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535
	assert.Greater(t, luminance, 0.5, "light theme overlay background should be bright")

	theme.Apply(theme.Presets["dark"]())
}

func TestApply_DarkChromeColors(t *testing.T) {
	theme.Apply(theme.Presets["dark"]())

	r, g, b, _ := theme.ColorOverlayBg.RGBA()
	luminance := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535
	assert.Less(t, luminance, 0.5, "dark theme overlay background should be dark")
}

func TestEntityColor_Deterministic(t *testing.T) {
	theme.Apply(theme.Presets["dark"]())

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
	theme.Apply(theme.Presets["dark"]())

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
