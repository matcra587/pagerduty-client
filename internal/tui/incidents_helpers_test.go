package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVisibleColumnIndices(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		wantIdx []int
	}{
		{"narrow <80 shows Sev Title Created", 60, []int{1, 2, 5}},
		{"medium 80-119 adds Assignees", 100, []int{1, 2, 4, 5}},
		{"wide 120-159 adds Service and Updated", 140, []int{1, 2, 3, 4, 5, 6}},
		{"full 160+ shows all including prefix", 180, []int{0, 1, 2, 3, 4, 5, 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := visibleColumnIndices(tt.width)
			assert.Equal(t, tt.wantIdx, got)
		})
	}
}

func TestLayoutColumnsBreakpoints(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		wantCols []string
	}{
		{"narrow <80", 60, []string{"Sev.", "Title", "Created"}},
		{"medium 80-119", 100, []string{"Sev.", "Title", "Assignees", "Created"}},
		{"wide 120-159", 140, []string{"Sev.", "Title", "Service", "Assignees", "Created", "Updated"}},
		{"full 160+", 180, []string{"", "Sev.", "Title", "Service", "Assignees", "Created", "Updated"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			widths := layoutColumns(tt.width, incidentColumns)
			var visible []string
			for i, w := range widths {
				if w > 0 {
					visible = append(visible, incidentColumns[i].header)
				}
			}
			assert.Equal(t, tt.wantCols, visible)
		})
	}
}

func TestLayoutColumnsWidthsSumFit(t *testing.T) {
	for _, width := range []int{60, 80, 100, 120, 140, 160, 180, 200} {
		t.Run("", func(t *testing.T) {
			widths := layoutColumns(width, incidentColumns)
			sum := 0
			visibleCount := 0
			for _, w := range widths {
				if w > 0 {
					sum += w
					visibleCount++
				}
			}
			separators := max(0, visibleCount-1)
			total := sum + separators
			require.LessOrEqual(t, total, width,
				"width=%d sum=%d separators=%d total=%d", width, sum, separators, total)
		})
	}
}

func TestLayoutColumnsFlexMinimums(t *testing.T) {
	for _, width := range []int{60, 80, 100, 120, 140, 160, 200} {
		widths := layoutColumns(width, incidentColumns)
		for i, w := range widths {
			if w == 0 {
				continue
			}
			c := incidentColumns[i]
			if c.width == 0 && c.min > 0 {
				assert.GreaterOrEqual(t, w, c.min,
					"width=%d col=%q got %d want >= %d", width, c.header, w, c.min)
			}
		}
	}
}

func TestLayoutColumnsHideService(t *testing.T) {
	for _, width := range []int{60, 100, 140, 180} {
		widths := layoutColumns(width, incidentColumns, colService)
		assert.Equal(t, 0, widths[colService],
			"width=%d: Service column should be hidden", width)
	}
}

func TestTruncate_NegativeWidth(t *testing.T) {
	assert.NotPanics(t, func() {
		result := truncate("hello world", -5)
		assert.Empty(t, result)
	})
}

func TestTruncate_ZeroWidth(t *testing.T) {
	assert.Empty(t, truncate("hello", 0))
}

func TestTruncate_ResultMatchesTargetWidth(t *testing.T) {
	result := truncate("hello world", 5)
	assert.Len(t, []rune(result), 5)
	assert.Equal(t, "hell…", result)
}

func TestTruncate_NoTruncationNeeded(t *testing.T) {
	result := truncate("hello", 10)
	assert.Equal(t, "hello", result)
}

func TestLayoutColumnsBreakpointBoundaries(t *testing.T) {
	tests := []struct {
		width       int
		wantVisible int
	}{
		{79, 3},
		{80, 4},
		{119, 4},
		{120, 6},
		{159, 6},
		{160, 7},
	}
	for _, tt := range tests {
		widths := layoutColumns(tt.width, incidentColumns)
		visible := 0
		for _, w := range widths {
			if w > 0 {
				visible++
			}
		}
		assert.Equal(t, tt.wantVisible, visible,
			"width=%d: expected %d visible columns, got %d", tt.width, tt.wantVisible, visible)
	}
}
