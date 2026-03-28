package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/service"
	"github.com/spf13/cobra"
)

var mcpStatusFormat string

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Inspect local MCP integration diagnostics",
	Long: strings.TrimSpace(`
Inspect HealthExport MCP integration status and local diagnostics.

Use this command to confirm that the local host configuration is ready for the
packaged Claude Desktop extension and future MCP-related tooling.
`),
	Example: strings.TrimSpace(`
  # Print a human-readable MCP status summary
  he mcp status

  # Print stable machine-readable diagnostics for local tooling
  he mcp status --format json
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var mcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show local MCP compatibility and auth diagnostics",
	Long: strings.TrimSpace(`
Show whether the current host machine is ready for the HealthExport MCP server.

The JSON output is a stable machine-readable contract for local diagnostics,
packaging support, and extension troubleshooting.
`),
	Example: strings.TrimSpace(`
  # Review host readiness for the MCP extension
  he mcp status

  # Emit machine-readable JSON
  he mcp status --format json
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPStatus(mcpStatusFormat, cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	mcpStatusCmd.Flags().StringVar(&mcpStatusFormat, "format", "", "Output format: json or text (defaults to text)")
	_ = mcpStatusCmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions([]cobra.Completion{"json", "text"}, cobra.ShellCompDirectiveNoFileComp))

	mcpCmd.AddCommand(mcpStatusCmd)
	rootCmd.AddCommand(mcpCmd)
}

func runMCPStatus(format string, stdout, stderr io.Writer) error {
	status, err := service.GetStatus(service.Options{
		AccountKey: accountKey,
		APIURL:     apiURL,
	}, version)
	if err != nil {
		return err
	}

	if strings.EqualFold(strings.TrimSpace(format), "json") {
		body, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(stdout, "%s\n", body)
		return err
	}

	if trimmed := strings.TrimSpace(format); trimmed != "" && !strings.EqualFold(trimmed, "text") {
		return exitError(fmt.Errorf("unsupported output format %q", format), 4)
	}

	fmt.Fprintf(stderr, "HealthExport MCP status\n")
	fmt.Fprintf(stderr, "  he version: %s\n", status.HEVersion)
	fmt.Fprintf(stderr, "  authenticated: %t\n", status.Authenticated)
	if status.AuthSource != "" {
		fmt.Fprintf(stderr, "  auth source: %s\n", status.AuthSource)
	}
	fmt.Fprintf(stderr, "  config path: %s\n", status.ConfigPath)
	fmt.Fprintf(stderr, "  api url: %s\n", status.APIURL)
	if !status.Authenticated {
		fmt.Fprintln(stderr, "  next step: run 'he auth login' on this host machine")
	}

	return nil
}
