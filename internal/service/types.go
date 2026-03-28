package service

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/typemap"
)

var ValidTypeCategories = []string{"aggregated", "record", "workout"}

func ListHealthTypes(opts Options, category string) ([]api.HealthType, error) {
	normalizedCategory, err := ValidateTypeCategory(category)
	if err != nil {
		return nil, err
	}

	resp, err := opts.APIClient().FetchHealthTypes()
	if err != nil {
		return nil, err
	}

	resolver := typemap.NewTypeResolver(resp)
	healthTypes := resolver.AllTypes()
	if normalizedCategory != "" {
		healthTypes = resolver.FilterByCategory(normalizedCategory)
	}

	slices.SortFunc(healthTypes, func(a, b api.HealthType) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return healthTypes, nil
}

func ValidateTypeCategory(category string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(category))
	if normalized == "" {
		return "", nil
	}

	for _, validCategory := range ValidTypeCategories {
		if normalized == validCategory {
			return normalized, nil
		}
	}

	return "", fmt.Errorf("%w: invalid category %q (valid categories: %s)", ErrInvalidInput, category, strings.Join(ValidTypeCategories, ", "))
}
