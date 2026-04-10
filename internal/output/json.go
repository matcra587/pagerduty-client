package output

import (
	"encoding/json"
	"io"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/gechr/clib/theme"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/compact"
)

// chromaStyle maps a clib theme preset name to a chroma style.
func chromaStyle(name string) string {
	switch name {
	case "dracula":
		return "dracula"
	case "catppuccin-latte":
		return "catppuccin-latte"
	case "catppuccin-frappe":
		return "catppuccin-frappe"
	case "catppuccin-macchiato":
		return "catppuccin-macchiato"
	case "catppuccin-mocha":
		return "catppuccin-mocha"
	case "monochrome":
		return "bw"
	default:
		return "monokai"
	}
}

// RenderJSON writes data as indented JSON to w. When th is non-nil,
// JSON output is syntax-highlighted using the theme's chroma style.
func RenderJSON(w io.Writer, data any, th *theme.Theme) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if th != nil {
		return quick.Highlight(w, string(b)+"\n", "json", "terminal256", chromaStyle(th.String()))
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

// RenderAgentJSON wraps data in an agent envelope and writes compact JSON to w.
// Pass nil for metadata when the response has no pagination.
func RenderAgentJSON(w io.Writer, command string, resource compact.Resource, data any, metadata *agent.Metadata, hints []string) error {
	compacted := compact.Compact(data)
	if rw, ok := compact.WeightsForResource(resource); ok {
		compacted = compact.BudgetSelect(compacted, rw)
	}
	env := agent.Success(command, compacted, metadata, hints)
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}
