package cmd

import (
	"errors"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/output"
	"github.com/TParizek/healthexport_cli/internal/service"
	"github.com/spf13/cobra"
)

var (
	typesFormat   string
	typesCategory string
)

var typesCmd = &cobra.Command{
	Use:   "types",
	Short: "List available health data types",
	Long: strings.TrimSpace(`
List the health data types exposed by the API.

Use this command to discover numeric IDs, canonical names, and which types are
available for aggregation. Authentication is not required.
`),
	Example: strings.TrimSpace(`
  # List all types
  he types

  # List only aggregatable types (for use with --aggregate)
  he types --category aggregated

  # JSON output
  he types --format json
`),
	Args: cobra.NoArgs,
	RunE: runTypes,
}

func init() {
	typesCmd.Flags().StringVar(&typesFormat, "format", "", "Output format for the type list: csv or json (defaults to config)")
	typesCmd.Flags().StringVar(&typesCategory, "category", "", "Filter by category: aggregated, record, or workout")
	_ = typesCmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions([]cobra.Completion{"csv", "json"}, cobra.ShellCompDirectiveNoFileComp))
	_ = typesCmd.RegisterFlagCompletionFunc("category", cobra.FixedCompletions([]cobra.Completion{"aggregated", "record", "workout"}, cobra.ShellCompDirectiveNoFileComp))

	rootCmd.AddCommand(typesCmd)
}

func runTypes(cmd *cobra.Command, args []string) error {
	healthTypes, err := service.ListHealthTypes(service.Options{APIURL: apiURL}, typesCategory)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			return exitError(err, 4)
		}

		return err
	}

	formatter, err := output.NewFormatter(resolveOutputFormat(typesFormat))
	if err != nil {
		return exitError(err, 4)
	}

	return formatter.FormatTypes(cmd.OutOrStdout(), healthTypes)
}
