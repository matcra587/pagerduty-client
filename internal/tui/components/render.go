package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// resetBg is the SGR 49 escape that restores the terminal's default background.
const resetBg = "\x1b[49m"

// PersistBgFull right-pads the line with spaces to the given width
// then applies PersistBg so the background spans the full terminal row.
func PersistBgFull(line, bg string, width int) string {
	lineW := lipgloss.Width(line)
	if lineW < width {
		line += strings.Repeat(" ", width-lineW)
	}
	return PersistBg(line, bg)
}

func PersistBg(line, bg string) string {
	if bg == "" {
		return line
	}
	var b strings.Builder
	b.WriteString(bg)
	for i := 0; i < len(line); i++ {
		if line[i] == '\x1b' && i+1 < len(line) && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && ((line[j] >= '0' && line[j] <= '9') || line[j] == ';') {
				j++
			}
			if j < len(line) && line[j] == 'm' {
				j++
				b.WriteString(line[i:j])
				b.WriteString(bg)
				i = j - 1
				continue
			}
		}
		b.WriteByte(line[i])
	}
	b.WriteString(resetBg)
	return b.String()
}

// ColorToANSIBg converts a colour to an ANSI 24-bit background escape sequence.
func ColorToANSIBg(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r>>8, g>>8, b>>8)
}
