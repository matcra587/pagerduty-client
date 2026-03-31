// Package output handles all output formatting for the pdc CLI.
// It determines the output format based on agent mode, explicit flags
// and TTY status, then renders data accordingly.
package output

// FormatType represents the output format to use.
type FormatType string

const (
	// FormatAgentJSON outputs JSON wrapped in an agent envelope.
	FormatAgentJSON FormatType = "agent-json"
	// FormatJSON outputs syntax-highlighted JSON on a TTY, plain JSON otherwise.
	FormatJSON FormatType = "json"
	// FormatTable outputs a styled table on a TTY.
	FormatTable FormatType = "table"
	// FormatPlainTable outputs an unstyled table without colour.
	FormatPlainTable FormatType = "plain-table"
)

// FormatOpts carries the inputs used to determine output format.
type FormatOpts struct {
	AgentMode bool
	Format    string
	IsTTY     bool
}

// DetectFormat returns the appropriate FormatType given opts.
// Priority: agent mode > explicit format flag > TTY detection.
func DetectFormat(opts FormatOpts) FormatType {
	if opts.AgentMode {
		return FormatAgentJSON
	}

	switch opts.Format {
	case "json":
		return FormatJSON
	case "table":
		if opts.IsTTY {
			return FormatTable
		}
		return FormatPlainTable
	}

	if opts.IsTTY {
		return FormatTable
	}
	return FormatPlainTable
}
