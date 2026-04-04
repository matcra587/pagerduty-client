package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no control characters",
			input: "CPU usage high on web-01",
			want:  "CPU usage high on web-01",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "preserves tabs and newlines",
			input: "line one\n\tindented",
			want:  "line one\n\tindented",
		},
		{
			name:  "preserves carriage return",
			input: "line\r\n",
			want:  "line\r\n",
		},
		{
			name:  "replaces ESC with caret",
			input: "Escape \x1b[2J\x1b[H pwned",
			want:  "Escape ^[[2J^[[H pwned",
		},
		{
			name:  "replaces BEL",
			input: "alert\x07done",
			want:  "alert^Gdone",
		},
		{
			name:  "replaces NUL",
			input: "before\x00after",
			want:  "before^@after",
		},
		{
			name:  "replaces C1 control character",
			input: "text\xc2\x9bmore",
			want:  "text^[more",
		},
		{
			name:  "mixed safe and unsafe",
			input: "Title: \x1b]0;hacked\x07 rest\nline two",
			want:  "Title: ^[]0;hacked^G rest\nline two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Sanitize(tt.input))
		})
	}
}

func TestSanitize_MalformedUTF8(t *testing.T) {
	t.Parallel()
	result := Sanitize("valid\xff\xfeinvalid")
	assert.NotContains(t, result, "\xff")
	assert.NotContains(t, result, "\xfe")
	assert.Contains(t, result, "valid")
	assert.Contains(t, result, "invalid")
}

func TestStripNonPrintable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "preserves normal text",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "preserves tabs and newlines",
			input: "line\n\ttab\r\n",
			want:  "line\n\ttab\r\n",
		},
		{
			name:  "strips control characters",
			input: "before\x1b[2Jafter",
			want:  "before[2Jafter",
		},
		{
			name:  "drops invalid UTF-8",
			input: "good\xff\xfebad",
			want:  "goodbad",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stripNonPrintable(tt.input))
		})
	}
}
