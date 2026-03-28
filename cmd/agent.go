package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	clib "github.com/gechr/clib/cli/cobra"
	"github.com/matcra587/pagerduty-client/internal/agent"
	"github.com/matcra587/pagerduty-client/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var agentCmd = &cobra.Command{
	Use:     "agent",
	Short:   "Agent discovery subcommands",
	Long:    "Subcommands for AI agent discovery: schema and guides.",
	GroupID: "config",
}

type commandSchema struct {
	Use      string          `json:"use"`
	Short    string          `json:"short"`
	Long     string          `json:"long,omitempty"`
	Flags    []flagSchema    `json:"flags,omitempty"`
	Commands []commandSchema `json:"commands,omitempty"`
}

type flagSchema struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Type      string `json:"type"`
	Default   string `json:"default,omitempty"`
	Usage     string `json:"usage,omitempty"`
}

func buildSchema(cmd *cobra.Command, compact bool) commandSchema {
	s := commandSchema{
		Use:   cmd.Use,
		Short: cmd.Short,
	}
	if !compact {
		s.Long = cmd.Long
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		fs := flagSchema{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Type:      f.Value.Type(),
			Default:   f.DefValue,
		}
		if !compact {
			fs.Usage = f.Usage
		}
		s.Flags = append(s.Flags, fs)
	})

	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		s.Commands = append(s.Commands, buildSchema(sub, compact))
	}

	return s
}

var agentSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Output machine-readable JSON schema of all pdc commands",
	Long: `Walk the pdc command tree and output a machine-readable JSON description
of every command, its flags and subcommands. AI agents can use this for
capability discovery without reading external documentation.

Use --compact to strip descriptions and examples for smaller context windows.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		compact, _ := cmd.Flags().GetBool("compact")
		schema := buildSchema(cmd.Root(), compact)

		det := AgentFromContext(cmd)
		if det.Active {
			return output.RenderAgentJSON(os.Stdout, "agent schema", output.ResourceNone, schema, nil, nil)
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(schema)
	},
}

var agentGuideCmd = &cobra.Command{
	Use:       "guide <name>",
	Short:     "Print a domain-specific markdown guide for AI agents",
	Long:      "Print an embedded markdown guide covering workflows and best practices for a PagerDuty domain.\nAvailable guides: " + joinGuideNames(),
	Args:      cobra.ExactArgs(1),
	ValidArgs: agent.GuideNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		content, err := agent.Guide(args[0])
		if err != nil {
			return err
		}
		_, _ = fmt.Fprint(os.Stdout, content)
		return nil
	},
}

func joinGuideNames() string {
	return strings.Join(agent.GuideNames, ", ")
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentSchemaCmd)
	agentCmd.AddCommand(agentGuideCmd)

	agentSchemaCmd.Flags().Bool("compact", false, "Strip descriptions and examples for smaller output")
	clib.Extend(agentSchemaCmd.Flags().Lookup("compact"), clib.FlagExtra{
		Group: "Output",
		Terse: "compact output",
	})
}
