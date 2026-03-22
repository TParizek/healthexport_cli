package typemap

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/api"
)

var ErrUnknownType = errors.New("unknown type")

type UnknownTypeError struct {
	Input       string
	Suggestions []string
}

func (e *UnknownTypeError) Error() string {
	if e == nil {
		return ErrUnknownType.Error()
	}

	input := strings.TrimSpace(e.Input)
	if input == "" {
		input = "<empty>"
	}

	if len(e.Suggestions) == 0 {
		return fmt.Sprintf("%s %q", ErrUnknownType.Error(), input)
	}

	return fmt.Sprintf("%s %q; did you mean: %s", ErrUnknownType.Error(), input, strings.Join(e.Suggestions, ", "))
}

func (e *UnknownTypeError) Unwrap() error {
	return ErrUnknownType
}

type TypeResolver struct {
	types      []api.HealthType
	aggregated map[int]bool
}

type suggestionCandidate struct {
	name       string
	normalized string
	distance   int
	order      int
}

func NewTypeResolver(resp *api.HealthTypesResponse) *TypeResolver {
	r := &TypeResolver{
		aggregated: make(map[int]bool),
	}

	if resp == nil {
		return r
	}

	r.appendSections("aggregated", resp.Aggregated)
	r.appendSections("record", resp.Record)
	r.appendSections("workout", resp.Workout)

	return r
}

func (r *TypeResolver) ResolveType(input string) ([]api.HealthType, error) {
	trimmed := strings.TrimSpace(input)

	if typeID, err := strconv.Atoi(trimmed); err == nil {
		for _, healthType := range r.types {
			if healthType.ID == typeID {
				return []api.HealthType{healthType}, nil
			}
		}

		return nil, &UnknownTypeError{Input: input}
	}

	normalized := normalize(trimmed)
	matches := make([]api.HealthType, 0)

	for _, healthType := range r.types {
		if normalize(healthType.Name) == normalized {
			matches = append(matches, healthType)
		}
	}

	if len(matches) > 0 {
		return matches, nil
	}

	return nil, &UnknownTypeError{
		Input:       input,
		Suggestions: r.Suggest(input, 3),
	}
}

func (r *TypeResolver) IsAggregatable(typeID int) bool {
	return r.aggregated[typeID]
}

func (r *TypeResolver) AllTypes() []api.HealthType {
	return slices.Clone(r.types)
}

func (r *TypeResolver) FilterByCategory(category string) []api.HealthType {
	normalizedCategory := strings.ToLower(strings.TrimSpace(category))
	if normalizedCategory == "" {
		return nil
	}

	filtered := make([]api.HealthType, 0)
	for _, healthType := range r.types {
		if strings.EqualFold(healthType.Category, normalizedCategory) {
			filtered = append(filtered, healthType)
		}
	}

	return filtered
}

func (r *TypeResolver) Suggest(input string, n int) []string {
	if n <= 0 {
		return nil
	}

	normalizedInput := normalize(input)
	if normalizedInput == "" {
		return nil
	}

	unique := make([]suggestionCandidate, 0)
	seen := make(map[string]struct{})
	for _, healthType := range r.types {
		normalizedName := normalize(healthType.Name)
		if _, ok := seen[normalizedName]; ok {
			continue
		}

		seen[normalizedName] = struct{}{}
		unique = append(unique, suggestionCandidate{
			name:       healthType.Name,
			normalized: normalizedName,
			distance:   levenshtein(normalizedInput, normalizedName),
			order:      len(unique),
		})
	}

	substrings := make([]suggestionCandidate, 0)
	ranked := slices.Clone(unique)
	for _, item := range unique {
		if strings.Contains(item.normalized, normalizedInput) {
			substrings = append(substrings, item)
		}
	}

	sortCandidates(substrings)
	sortCandidates(ranked)

	suggestions := make([]string, 0, n)
	added := make(map[string]struct{})
	for _, group := range [][]suggestionCandidate{substrings, ranked} {
		for _, item := range group {
			if _, ok := added[item.normalized]; ok {
				continue
			}

			added[item.normalized] = struct{}{}
			suggestions = append(suggestions, item.name)
			if len(suggestions) == n {
				return suggestions
			}
		}
	}

	return suggestions
}

func (r *TypeResolver) appendSections(category string, sections []api.HealthTypeSection) {
	for _, section := range sections {
		for _, healthType := range section.Types {
			flattened := healthType
			flattened.Category = category
			r.types = append(r.types, flattened)

			if category == "aggregated" {
				r.aggregated[flattened.ID] = true
			}
		}
	}
}

func normalize(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", " ")
	return strings.Join(strings.Fields(s), " ")
}

func sortCandidates(candidates []suggestionCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].distance != candidates[j].distance {
			return candidates[i].distance < candidates[j].distance
		}

		if candidates[i].name != candidates[j].name {
			return candidates[i].name < candidates[j].name
		}

		return candidates[i].order < candidates[j].order
	})
}

func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)

	if len(ar) == 0 {
		return len(br)
	}

	if len(br) == 0 {
		return len(ar)
	}

	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}

			insertCost := curr[j-1] + 1
			deleteCost := prev[j] + 1
			replaceCost := prev[j-1] + cost

			curr[j] = min(insertCost, deleteCost, replaceCost)
		}

		prev, curr = curr, prev
	}

	return prev[len(br)]
}
