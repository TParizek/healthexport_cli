package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TParizek/healthexport_cli/internal/aggregator"
	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/TParizek/healthexport_cli/internal/crypto"
	"github.com/TParizek/healthexport_cli/internal/output"
	"github.com/TParizek/healthexport_cli/internal/typemap"
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
	accountKey, _, err := auth.Resolve(accountKey)
	if err != nil {
		return err
	}

	dateFrom, err := parseDate(dataFrom)
	if err != nil {
		return exitError(err, 4)
	}

	dateTo, err := parseDate(dataTo)
	if err != nil {
		return exitError(err, 4)
	}

	var period *aggregator.Period
	if strings.TrimSpace(dataAggregate) != "" {
		parsedPeriod, err := aggregator.ParsePeriod(dataAggregate)
		if err != nil {
			return exitError(err, 4)
		}

		period = &parsedPeriod
	}

	client := api.NewClient(resolveAPIURL())

	typeIDs, typeNames, err := resolveTypes(client, dataTypes, period)
	if err != nil {
		if errors.Is(err, typemap.ErrUnknownType) || errors.Is(err, aggregator.ErrNotAggregatable) {
			return exitError(err, 4)
		}

		return err
	}

	encrypted, err := client.FetchEncryptedData(accountKey.UID, typeIDs, dateFrom, dateTo)
	if err != nil {
		return err
	}

	if dataRaw {
		enrichEncryptedTypeNames(encrypted, typeNames)
		return output.JSONFormatter{}.FormatRawData(cmd.OutOrStdout(), encrypted)
	}

	if len(typeNames) == 0 {
		typeNames = fetchTypeNamesBestEffort(client, typeIDs)
	}

	decrypted, err := crypto.DecryptRecords(encrypted, accountKey.DecryptionKey)
	if err != nil {
		return err
	}
	enrichDecryptedTypeNames(decrypted, typeNames)

	format := resolveOutputFormat(dataFormat)
	formatter, err := output.NewFormatter(format)
	if err != nil {
		return exitError(err, 4)
	}

	if period != nil {
		aggregated, err := aggregatePackages(decrypted, *period)
		if err != nil {
			return err
		}

		return formatter.FormatAggregatedData(cmd.OutOrStdout(), aggregated)
	}

	return formatter.FormatData(cmd.OutOrStdout(), decrypted)
}

func parseDate(s string) (string, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", fmt.Errorf("date is required (expected YYYY-MM-DD or RFC3339)")
	}

	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.Format(time.RFC3339), nil
	}

	if parsed, err := time.Parse("2006-01-02", trimmed); err == nil {
		return parsed.UTC().Format(time.RFC3339), nil
	}

	return "", fmt.Errorf("invalid date %q (expected YYYY-MM-DD or RFC3339)", s)
}

func resolveTypes(client *api.Client, inputs []string, aggregate *aggregator.Period) ([]int, map[int]string, error) {
	ids, allNumeric, err := parseTypeInputs(inputs)
	if err != nil {
		return nil, nil, err
	}

	if allNumeric {
		if aggregate == nil {
			return ids, nil, nil
		}

		resp, err := client.FetchHealthTypes()
		if err != nil {
			return nil, nil, err
		}

		resolver := typemap.NewTypeResolver(resp)
		names, err := validateResolvedIDs(resolver, ids, aggregate)
		if err != nil {
			return nil, nil, err
		}

		return ids, names, nil
	}

	resp, err := client.FetchHealthTypes()
	if err != nil {
		return nil, nil, err
	}

	resolver := typemap.NewTypeResolver(resp)
	resolvedIDs := make([]int, 0, len(inputs))
	typeNames := make(map[int]string)

	for _, input := range inputs {
		matches, err := resolver.ResolveType(input)
		if err != nil {
			return nil, nil, err
		}

		if aggregate != nil {
			matches = filterAggregatableMatches(matches, resolver)
			if len(matches) == 0 {
				return nil, nil, fmt.Errorf("%w: %s", aggregator.ErrNotAggregatable, strings.TrimSpace(input))
			}
		}

		for _, match := range matches {
			if aggregate != nil && !resolver.IsAggregatable(match.ID) {
				return nil, nil, fmt.Errorf("%w: %s (%d)", aggregator.ErrNotAggregatable, match.Name, match.ID)
			}

			if _, ok := typeNames[match.ID]; !ok {
				resolvedIDs = append(resolvedIDs, match.ID)
			}
			typeNames[match.ID] = match.Name
		}
	}

	return resolvedIDs, typeNames, nil
}

func parseTypeInputs(inputs []string) ([]int, bool, error) {
	ids := make([]int, 0, len(inputs))
	seen := make(map[int]struct{})
	allNumeric := true

	for _, input := range inputs {
		trimmed := strings.TrimSpace(input)
		typeID, err := strconv.Atoi(trimmed)
		if err != nil {
			allNumeric = false
			continue
		}

		if _, ok := seen[typeID]; ok {
			continue
		}

		seen[typeID] = struct{}{}
		ids = append(ids, typeID)
	}

	if allNumeric {
		return ids, true, nil
	}

	return nil, false, nil
}

func validateResolvedIDs(resolver *typemap.TypeResolver, ids []int, aggregate *aggregator.Period) (map[int]string, error) {
	typeNames := make(map[int]string, len(ids))

	for _, typeID := range ids {
		matches, err := resolver.ResolveType(strconv.Itoa(typeID))
		if err != nil {
			return nil, err
		}

		match := matches[0]
		typeNames[typeID] = match.Name

		if aggregate != nil && !resolver.IsAggregatable(typeID) {
			return nil, fmt.Errorf("%w: %s (%d)", aggregator.ErrNotAggregatable, match.Name, typeID)
		}
	}

	return typeNames, nil
}

func filterAggregatableMatches(matches []api.HealthType, resolver *typemap.TypeResolver) []api.HealthType {
	filtered := make([]api.HealthType, 0, len(matches))
	for _, match := range matches {
		if resolver.IsAggregatable(match.ID) {
			filtered = append(filtered, match)
		}
	}

	return filtered
}

func aggregatePackages(packages []api.DecryptedPackage, period aggregator.Period) ([]api.AggregatedPackage, error) {
	aggregated := make([]api.AggregatedPackage, 0, len(packages))

	for _, pkg := range packages {
		outPkg := api.AggregatedPackage{
			Type:     pkg.Type,
			TypeName: pkg.TypeName,
			Data:     make([]api.AggregatedUnitGroup, 0, len(pkg.Data)),
		}

		for _, group := range pkg.Data {
			records, _, err := aggregator.Aggregate(group.Records, period)
			if err != nil {
				return nil, err
			}

			outPkg.Data = append(outPkg.Data, api.AggregatedUnitGroup{
				Units:   group.Units,
				Records: records,
			})
		}

		aggregated = append(aggregated, outPkg)
	}

	return aggregated, nil
}

func resolveAPIURL() string {
	if trimmed := strings.TrimSpace(apiURL); trimmed != "" {
		return trimmed
	}

	cfg, err := config.Load()
	if err == nil && strings.TrimSpace(cfg.APIURL) != "" {
		return strings.TrimSpace(cfg.APIURL)
	}

	return config.DefaultAPIURL
}

func resolveOutputFormat(flagValue string) string {
	if trimmed := strings.TrimSpace(flagValue); trimmed != "" {
		return trimmed
	}

	cfg, err := config.Load()
	if err == nil && strings.TrimSpace(cfg.Format) != "" {
		return strings.TrimSpace(cfg.Format)
	}

	return config.DefaultFormat
}

func fetchTypeNamesBestEffort(client *api.Client, typeIDs []int) map[int]string {
	resp, err := client.FetchHealthTypes()
	if err != nil {
		return map[int]string{}
	}

	resolver := typemap.NewTypeResolver(resp)
	names := make(map[int]string, len(typeIDs))
	for _, typeID := range typeIDs {
		matches, err := resolver.ResolveType(strconv.Itoa(typeID))
		if err != nil || len(matches) == 0 {
			continue
		}

		names[typeID] = matches[0].Name
	}

	return names
}

func enrichEncryptedTypeNames(packages []api.EncryptedPackage, typeNames map[int]string) {
	for i := range packages {
		packages[i].TypeName = typeNames[packages[i].Type]
	}
}

func enrichDecryptedTypeNames(packages []api.DecryptedPackage, typeNames map[int]string) {
	for i := range packages {
		packages[i].TypeName = typeNames[packages[i].Type]
	}
}

func enrichAggregatedTypeNames(packages []api.AggregatedPackage, typeNames map[int]string) {
	for i := range packages {
		packages[i].TypeName = typeNames[packages[i].Type]
	}
}
