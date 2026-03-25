package components

import (
	"fmt"
	"image/color"
	"strings"
)

// resetBg is the SGR 49 escape that restores the terminal's default background.
const resetBg = "\x1b[49m"

// PersistBg re-applies an ANSI background escape after every SGR sequence
// in line, preventing lipgloss Render resets from clearing the container's
// background. Inspired by prl's injectLineBackground.
//
// Safe for multi-byte UTF-8: ANSI escapes use only ASCII bytes (0x1B,
// 0x5B, 0x30-0x39, 0x3B, 0x6D). UTF-8 continuation bytes (0x80-0xBF)
// never match these, so byte-level scanning cannot split a codepoint.
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
