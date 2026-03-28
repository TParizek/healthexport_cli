package cmd

import (
	"errors"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/aggregator"
	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/output"
	"github.com/TParizek/healthexport_cli/internal/service"
	"github.com/spf13/cobra"
)

var (
	dataTypes     []string
	dataFrom      string
	dataTo        string
	dataFormat    string
	dataRaw       bool
	dataAggregate string
)

var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "Fetch and decrypt health records",
	Long: strings.TrimSpace(`
Fetch encrypted health records and decrypt them locally.

Use --type with one or more type names or numeric IDs, then choose a date
range with --from and --to. Type names are resolved from the API at runtime,
so --type does not support shell completion.
`),
	Example: strings.TrimSpace(`
  # Fetch step count for January 2024
  he data --type step_count --from 2024-01-01 --to 2024-01-31

  # Fetch by numeric type ID
  he data --type 9 --from 2024-01-01 --to 2024-01-31

  # Multiple types, JSON output
  he data -t step_count -t body_mass -f 2024-01-01 -T 2024-01-31 --format json

  # Aggregate steps by day
  he data -t step_count -f 2024-01-01 -T 2024-01-31 --aggregate day

  # View raw encrypted data
  he data -t 9 -f 2024-01-01 -T 2024-01-31 --raw
`),
	Args: cobra.NoArgs,
	RunE: runData,
}

func init() {
	dataCmd.Flags().StringSliceVarP(&dataTypes, "type", "t", nil, "Health data type name or numeric ID (repeatable; no shell completion because values come from the API)")
	dataCmd.Flags().StringVarP(&dataFrom, "from", "f", "", "Start date, in YYYY-MM-DD or RFC3339 format")
	dataCmd.Flags().StringVarP(&dataTo, "to", "T", "", "End date, in YYYY-MM-DD or RFC3339 format")
	dataCmd.Flags().StringVar(&dataFormat, "format", "", "Output format for decrypted records: csv or json (defaults to config)")
	dataCmd.Flags().BoolVar(&dataRaw, "raw", false, "Output encrypted records as JSON without local decryption")
	dataCmd.Flags().StringVarP(&dataAggregate, "aggregate", "a", "", "Aggregate period for compatible types: day, week, month, or year")

	_ = dataCmd.MarkFlagRequired("type")
	_ = dataCmd.MarkFlagRequired("from")
	_ = dataCmd.MarkFlagRequired("to")
	_ = dataCmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions([]cobra.Completion{"csv", "json"}, cobra.ShellCompDirectiveNoFileComp))
	_ = dataCmd.RegisterFlagCompletionFunc("aggregate", cobra.FixedCompletions([]cobra.Completion{"day", "week", "month", "year"}, cobra.ShellCompDirectiveNoFileComp))

	rootCmd.AddCommand(dataCmd)
}

func runData(cmd *cobra.Command, args []string) error {
	opts := service.Options{
		AccountKey: accountKey,
		APIURL:     apiURL,
	}

	result, err := service.FetchHealthData(opts, service.FetchRequest{
		Types:     dataTypes,
		From:      dataFrom,
		To:        dataTo,
		Aggregate: dataAggregate,
		Raw:       dataRaw,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			return exitError(err, 4)
		default:
			return err
		}
	}

	if dataRaw {
		return output.JSONFormatter{}.FormatRawData(cmd.OutOrStdout(), result.Raw)
	}

	format := resolveOutputFormat(dataFormat)
	formatter, err := output.NewFormatter(format)
	if err != nil {
		return exitError(err, 4)
	}

	if strings.TrimSpace(dataAggregate) != "" {
		return formatter.FormatAggregatedData(cmd.OutOrStdout(), result.Aggregated)
	}

	return formatter.FormatData(cmd.OutOrStdout(), result.Decrypted)
}

func parseDate(s string) (string, error) {
	return service.ParseDate(s)
}

func resolveTypes(client *api.Client, inputs []string, aggregate *aggregator.Period) ([]int, map[int]string, error) {
	return service.ResolveTypes(client, inputs, aggregate)
}

func resolveOutputFormat(flagValue string) string {
	return service.Options{APIURL: apiURL}.ResolvedOutputFormat(flagValue)
}
