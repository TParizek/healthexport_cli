package service

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/TParizek/healthexport_cli/internal/aggregator"
	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/crypto"
	"github.com/TParizek/healthexport_cli/internal/typemap"
)

type FetchRequest struct {
	Types                   []string
	From                    string
	To                      string
	Aggregate               string
	Raw                     bool
	AllowPartialAggregation bool
}

type FetchResult struct {
	From            string                  `json:"from"`
	To              string                  `json:"to"`
	Aggregate       string                  `json:"aggregate,omitempty"`
	ResolvedTypeIDs []int                   `json:"resolved_type_ids,omitempty"`
	Warnings        []string                `json:"warnings,omitempty"`
	Raw             []api.EncryptedPackage  `json:"raw,omitempty"`
	Decrypted       []api.DecryptedPackage  `json:"results,omitempty"`
	Aggregated      []api.AggregatedPackage `json:"aggregated_results,omitempty"`
}

type resolvedTypes struct {
	IDs                      []int
	Names                    map[int]string
	AggregatableIDs          map[int]bool
	NonAggregatableTypeNames []string
}

type unknownHealthTypesError struct {
	Names []string
	IDs   []string
}

func (e *unknownHealthTypesError) Error() string {
	if e == nil {
		return typemap.ErrUnknownType.Error()
	}

	parts := make([]string, 0, 2)
	if len(e.Names) == 1 {
		parts = append(parts, fmt.Sprintf("unknown health type: %s", e.Names[0]))
	} else if len(e.Names) > 1 {
		parts = append(parts, fmt.Sprintf("unknown health types: %s", strings.Join(e.Names, ", ")))
	}

	if len(e.IDs) == 1 {
		parts = append(parts, fmt.Sprintf("unknown health type ID: %s", e.IDs[0]))
	} else if len(e.IDs) > 1 {
		parts = append(parts, fmt.Sprintf("unknown health type IDs: %s", strings.Join(e.IDs, ", ")))
	}

	if len(parts) == 0 {
		return typemap.ErrUnknownType.Error()
	}

	return strings.Join(parts, "; ")
}

func (e *unknownHealthTypesError) Unwrap() error {
	return typemap.ErrUnknownType
}

func FetchHealthData(opts Options, req FetchRequest) (*FetchResult, error) {
	if len(req.Types) == 0 {
		return nil, fmt.Errorf("%w: the 'types' parameter must contain at least one health type", ErrInvalidInput)
	}

	fromTime, dateFrom, err := parseDateValue(req.From)
	if err != nil {
		return nil, err
	}

	toTime, dateTo, err := parseDateValue(req.To)
	if err != nil {
		return nil, err
	}

	if fromTime.After(toTime) {
		fromTime, toTime = toTime, fromTime
		dateFrom, dateTo = dateTo, dateFrom
	}

	var period *aggregator.Period
	if strings.TrimSpace(req.Aggregate) != "" {
		parsedPeriod, err := aggregator.ParsePeriod(req.Aggregate)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
		}

		period = &parsedPeriod
	}

	client := opts.APIClient()
	resolved, err := resolveTypesDetailed(client, req.Types, period, req.AllowPartialAggregation)
	if err != nil {
		if errors.Is(err, typemap.ErrUnknownType) || errors.Is(err, aggregator.ErrNotAggregatable) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidInput, err)
		}

		return nil, err
	}

	accountKey, _, err := auth.ResolveWithConfigPath(opts.AccountKey, opts.ConfigPath)
	if err != nil {
		return nil, err
	}

	encrypted, err := client.FetchEncryptedData(accountKey.UID, resolved.IDs, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}

	result := &FetchResult{
		From:            dateFrom,
		To:              dateTo,
		ResolvedTypeIDs: resolved.IDs,
	}

	if req.Raw {
		enrichEncryptedTypeNames(encrypted, resolved.Names)
		result.Raw = encrypted
		return result, nil
	}

	typeNames := resolved.Names
	if len(typeNames) == 0 {
		typeNames = fetchTypeNamesBestEffort(client, resolved.IDs)
	}

	decrypted, err := crypto.DecryptRecords(encrypted, accountKey.DecryptionKey)
	if err != nil {
		return nil, err
	}
	enrichDecryptedTypeNames(decrypted, typeNames)

	if period != nil {
		aggregatablePackages, rawPackages := partitionPackagesByAggregation(decrypted, resolved.AggregatableIDs)

		if len(aggregatablePackages) > 0 {
			aggregated, err := aggregatePackages(aggregatablePackages, *period)
			if err != nil {
				return nil, err
			}

			enrichAggregatedTypeNames(aggregated, typeNames)
			result.Aggregate = string(*period)
			result.Aggregated = aggregated
		}

		if len(rawPackages) > 0 {
			result.Decrypted = rawPackages
		}

		if len(result.Aggregated) == 0 && len(result.Decrypted) == 0 {
			result.Aggregate = string(*period)
			result.Aggregated = []api.AggregatedPackage{}
		}

		result.Warnings = buildWarnings(combinedRecordCount(result.Decrypted, result.Aggregated), opts.EffectiveMaxRecordsWarningThreshold())
		if req.AllowPartialAggregation && len(resolved.NonAggregatableTypeNames) > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("aggregation was ignored for non-aggregatable types: %s", strings.Join(resolved.NonAggregatableTypeNames, ", ")))
		}

		return result, nil
	}

	result.Decrypted = decrypted
	result.Warnings = buildWarnings(decryptedRecordCount(decrypted), opts.EffectiveMaxRecordsWarningThreshold())
	return result, nil
}

func ParseDate(s string) (string, error) {
	_, formatted, err := parseDateValue(s)
	return formatted, err
}

func ResolveTypes(client *api.Client, inputs []string, aggregate *aggregator.Period) ([]int, map[int]string, error) {
	resolved, err := resolveTypesDetailed(client, inputs, aggregate, false)
	if err != nil {
		return nil, nil, err
	}

	return resolved.IDs, resolved.Names, nil
}

func parseDateValue(s string) (time.Time, string, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return time.Time{}, "", fmt.Errorf("%w: date is required (expected YYYY-MM-DD or RFC3339)", ErrInvalidInput)
	}

	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed, parsed.Format(time.RFC3339), nil
	}

	if parsed, err := time.Parse("2006-01-02", trimmed); err == nil {
		parsed = parsed.UTC()
		return parsed, parsed.Format(time.RFC3339), nil
	}

	return time.Time{}, "", fmt.Errorf("%w: invalid date %q (expected YYYY-MM-DD or RFC3339)", ErrInvalidInput, s)
}

func resolveTypesDetailed(client *api.Client, inputs []string, aggregate *aggregator.Period, allowPartialAggregation bool) (*resolvedTypes, error) {
	ids, allNumeric, err := parseTypeInputs(inputs)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}

	if allNumeric && aggregate == nil {
		return &resolvedTypes{IDs: ids}, nil
	}

	resp, err := client.FetchHealthTypes()
	if err != nil {
		return nil, err
	}

	resolver := typemap.NewTypeResolver(resp)
	if allNumeric {
		return resolveNumericTypes(resolver, ids, aggregate, allowPartialAggregation)
	}

	return resolveNamedTypes(resolver, inputs, aggregate, allowPartialAggregation)
}

func resolveNumericTypes(resolver *typemap.TypeResolver, ids []int, aggregate *aggregator.Period, allowPartialAggregation bool) (*resolvedTypes, error) {
	typeNames := make(map[int]string, len(ids))
	aggregatableIDs := make(map[int]bool, len(ids))
	unknownIDs := make([]string, 0)
	nonAggregatable := make([]string, 0)

	for _, typeID := range ids {
		matches, err := resolver.ResolveType(strconv.Itoa(typeID))
		if err != nil {
			if errors.Is(err, typemap.ErrUnknownType) {
				unknownIDs = append(unknownIDs, strconv.Itoa(typeID))
				continue
			}

			return nil, err
		}

		match := matches[0]
		typeNames[typeID] = match.Name
		aggregatableIDs[typeID] = resolver.IsAggregatable(typeID)

		if aggregate != nil && !allowPartialAggregation && !resolver.IsAggregatable(typeID) {
			nonAggregatable = append(nonAggregatable, match.Name)
		}
	}

	if len(unknownIDs) > 0 {
		return nil, &unknownHealthTypesError{IDs: uniqueSortedStrings(unknownIDs)}
	}

	if len(nonAggregatable) > 0 {
		return nil, fmt.Errorf("%w: types %s do not support aggregation. Remove the 'aggregate' parameter or use only aggregated-category types", aggregator.ErrNotAggregatable, quoteList(uniqueSortedStrings(nonAggregatable)))
	}

	return &resolvedTypes{
		IDs:                      ids,
		Names:                    typeNames,
		AggregatableIDs:          aggregatableIDs,
		NonAggregatableTypeNames: uniqueSortedStrings(nonAggregatableTypeNames(typeNames, aggregatableIDs)),
	}, nil
}

func resolveNamedTypes(resolver *typemap.TypeResolver, inputs []string, aggregate *aggregator.Period, allowPartialAggregation bool) (*resolvedTypes, error) {
	resolvedIDs := make([]int, 0, len(inputs))
	typeNames := make(map[int]string)
	aggregatableIDs := make(map[int]bool)
	unknownNames := make([]string, 0)
	unknownIDs := make([]string, 0)
	nonAggregatable := make([]string, 0)

	for _, input := range inputs {
		matches, err := resolver.ResolveType(input)
		if err != nil {
			if errors.Is(err, typemap.ErrUnknownType) {
				if _, parseErr := strconv.Atoi(strings.TrimSpace(input)); parseErr == nil {
					unknownIDs = append(unknownIDs, strings.TrimSpace(input))
				} else {
					unknownNames = append(unknownNames, strings.TrimSpace(input))
				}
				continue
			}

			return nil, err
		}

		selectedMatches := matches
		if aggregate != nil && !allowPartialAggregation {
			selectedMatches = filterAggregatableMatches(matches, resolver)
			if len(selectedMatches) == 0 {
				nonAggregatable = append(nonAggregatable, strings.TrimSpace(input))
				continue
			}
		}

		for _, match := range selectedMatches {
			if _, ok := typeNames[match.ID]; !ok {
				resolvedIDs = append(resolvedIDs, match.ID)
			}

			typeNames[match.ID] = match.Name
			aggregatableIDs[match.ID] = resolver.IsAggregatable(match.ID)
		}
	}

	if len(unknownNames) > 0 || len(unknownIDs) > 0 {
		return nil, &unknownHealthTypesError{
			Names: uniqueSortedStrings(unknownNames),
			IDs:   uniqueSortedStrings(unknownIDs),
		}
	}

	if len(nonAggregatable) > 0 {
		return nil, fmt.Errorf("%w: types %s do not support aggregation. Remove the 'aggregate' parameter or use only aggregated-category types", aggregator.ErrNotAggregatable, quoteList(uniqueSortedStrings(nonAggregatable)))
	}

	return &resolvedTypes{
		IDs:                      resolvedIDs,
		Names:                    typeNames,
		AggregatableIDs:          aggregatableIDs,
		NonAggregatableTypeNames: uniqueSortedStrings(nonAggregatableTypeNames(typeNames, aggregatableIDs)),
	}, nil
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

			if group.Units == "count" {
				roundAggregatedCountRecords(records)
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

func decryptedRecordCount(packages []api.DecryptedPackage) int {
	count := 0
	for _, pkg := range packages {
		for _, group := range pkg.Data {
			count += len(group.Records)
		}
	}

	return count
}

func aggregatedRecordCount(packages []api.AggregatedPackage) int {
	count := 0
	for _, pkg := range packages {
		for _, group := range pkg.Data {
			count += len(group.Records)
		}
	}

	return count
}

func combinedRecordCount(decrypted []api.DecryptedPackage, aggregated []api.AggregatedPackage) int {
	return decryptedRecordCount(decrypted) + aggregatedRecordCount(aggregated)
}

func buildWarnings(recordCount, threshold int) []string {
	if threshold <= 0 || recordCount <= threshold {
		return nil
	}

	return []string{
		fmt.Sprintf("response contains %d records; narrow the date range or use aggregation for smaller tool responses", recordCount),
	}
}

func partitionPackagesByAggregation(packages []api.DecryptedPackage, aggregatableIDs map[int]bool) ([]api.DecryptedPackage, []api.DecryptedPackage) {
	aggregatable := make([]api.DecryptedPackage, 0, len(packages))
	raw := make([]api.DecryptedPackage, 0, len(packages))

	for _, pkg := range packages {
		if aggregatableIDs[pkg.Type] {
			aggregatable = append(aggregatable, pkg)
			continue
		}

		raw = append(raw, pkg)
	}

	return aggregatable, raw
}

func roundAggregatedCountRecords(records []api.AggregatedRecord) {
	for i := range records {
		records[i].Value = math.Round(records[i].Value)
	}
}

func nonAggregatableTypeNames(typeNames map[int]string, aggregatableIDs map[int]bool) []string {
	names := make([]string, 0)
	for typeID, name := range typeNames {
		if !aggregatableIDs[typeID] {
			names = append(names, name)
		}
	}

	return names
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		if _, ok := seen[trimmed]; ok {
			continue
		}

		seen[trimmed] = struct{}{}
		unique = append(unique, trimmed)
	}

	slices.Sort(unique)
	return unique
}

func quoteList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	return strings.Join(quoted, ", ")
}
