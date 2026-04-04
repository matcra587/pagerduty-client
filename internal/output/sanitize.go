package output

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/cli/go-gh/v2/pkg/asciisanitizer"
	"github.com/gechr/clog"
	"golang.org/x/text/transform"
)

// Sanitize replaces C0 and C1 ASCII control characters with visible
// caret notation to prevent terminal escape injection. Tabs, newlines
// and carriage returns are preserved. On malformed UTF-8, non-printable
// bytes are stripped rather than passed through.
func Sanitize(s string) string {
	out, _, err := transform.String(&asciisanitizer.Sanitizer{}, s)
	if err != nil {
		clog.Debug().Err(err).Msg("sanitise failed, stripping non-printable bytes")
		return stripNonPrintable(s)
	}
	return out
}

// stripNonPrintable removes non-printable, non-space runes and replaces
// invalid UTF-8 sequences. Used as a fallback when the transformer fails.
func stripNonPrintable(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size < 2 {
			i++
			continue
		}
		if r == '\t' || r == '\n' || r == '\r' || !unicode.IsControl(r) {
			sb.WriteRune(r)
		}
		i += size
	}
	return sb.String()
}
