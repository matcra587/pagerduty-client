package output

import (
	"encoding/json"
	"io"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/compact"
)

// RenderJSON writes data as indented JSON to w. When isTTY is true,
// JSON output is syntax-highlighted for terminal display.
func RenderJSON(w io.Writer, data any, isTTY bool) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if isTTY {
		return quick.Highlight(w, string(b)+"\n", "json", "terminal256", "monokai")
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
