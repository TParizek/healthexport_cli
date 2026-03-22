package cmd

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/output"
	"github.com/TParizek/healthexport_cli/internal/typemap"
	"github.com/spf13/cobra"
)

var (
	typesFormat   string
	typesCategory string
)

var validTypeCategories = []string{"aggregated", "record", "workout"}

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
	category, err := validateTypeCategory(typesCategory)
	if err != nil {
		return exitError(err, 4)
	}

	client := api.NewClient(resolveAPIURL())
	resp, err := client.FetchHealthTypes()
	if err != nil {
		return err
	}

	resolver := typemap.NewTypeResolver(resp)
	healthTypes := resolver.AllTypes()
	if category != "" {
		healthTypes = resolver.FilterByCategory(category)
	}

	slices.SortFunc(healthTypes, func(a, b api.HealthType) int {
		return cmp.Compare(a.ID, b.ID)
	})

	formatter, err := output.NewFormatter(resolveOutputFormat(typesFormat))
	if err != nil {
		return exitError(err, 4)
	}

	return formatter.FormatTypes(cmd.OutOrStdout(), healthTypes)
}

func validateTypeCategory(category string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(category))
	if normalized == "" {
		return "", nil
	}

	for _, validCategory := range validTypeCategories {
		if normalized == validCategory {
			return normalized, nil
		}
	}

	return "", fmt.Errorf("invalid category %q (valid categories: %s)", category, strings.Join(validTypeCategories, ", "))
}
