package typemap

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/api"
)

func TestResolveTypeByNumericID(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	got, err := resolver.ResolveType("9")
	if err != nil {
		t.Fatalf("ResolveType() error = %v", err)
	}

	assertTypeIDs(t, got, 9)
	if got[0].Name != "Step count" {
		t.Fatalf("got name = %q, want %q", got[0].Name, "Step count")
	}
}

func TestResolveTypeUnknownNumericID(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	_, err := resolver.ResolveType("999")
	if err == nil {
		t.Fatal("ResolveType() error = nil, want unknown type error")
	}

	if !errors.Is(err, ErrUnknownType) {
		t.Fatalf("errors.Is(err, ErrUnknownType) = false, err = %v", err)
	}
}

func TestResolveTypeByNameVariants(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	testCases := []struct {
		input string
		want  []int
	}{
		{input: "step_count", want: []int{9}},
		{input: "Step count", want: []int{9}},
		{input: "STEP_COUNT", want: []int{9}},
		{input: "body mass", want: []int{0}},
		{input: "body_mass_index", want: []int{47}},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := resolver.ResolveType(tc.input)
			if err != nil {
				t.Fatalf("ResolveType() error = %v", err)
			}

			assertTypeIDs(t, got, tc.want...)
		})
	}
}

func TestResolveTypeReturnsMultipleMatches(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	got, err := resolver.ResolveType("heart_rate")
	if err != nil {
		t.Fatalf("ResolveType() error = %v", err)
	}

	assertTypeIDs(t, got, 6, 52)
	if got[0].Category != "aggregated" || got[1].Category != "record" {
		t.Fatalf("categories = [%q, %q], want [aggregated, record]", got[0].Category, got[1].Category)
	}
}

func TestResolveTypeUnknownIncludesSuggestions(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	_, err := resolver.ResolveType("nonexistent_type")
	if err == nil {
		t.Fatal("ResolveType() error = nil, want unknown type error")
	}

	if !errors.Is(err, ErrUnknownType) {
		t.Fatalf("errors.Is(err, ErrUnknownType) = false, err = %v", err)
	}

	var unknownErr *UnknownTypeError
	if !errors.As(err, &unknownErr) {
		t.Fatalf("errors.As(err, *UnknownTypeError) = false, err = %T", err)
	}

	if len(unknownErr.Suggestions) == 0 {
		t.Fatal("UnknownTypeError.Suggestions is empty, want suggestions")
	}
}

func TestResolveTypeTypoSuggestsClosestName(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	_, err := resolver.ResolveType("step_cunt")
	if err == nil {
		t.Fatal("ResolveType() error = nil, want unknown type error")
	}

	if !strings.Contains(err.Error(), "Step count") {
		t.Fatalf("error = %q, want Step count suggestion", err.Error())
	}
}

func TestIsAggregatable(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	if !resolver.IsAggregatable(9) {
		t.Fatal("IsAggregatable(9) = false, want true")
	}

	if resolver.IsAggregatable(0) {
		t.Fatal("IsAggregatable(0) = true, want false")
	}
}

func TestFilterByCategory(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	aggregated := resolver.FilterByCategory("aggregated")
	assertTypeIDs(t, aggregated, 5, 6, 9)

	record := resolver.FilterByCategory("record")
	assertTypeIDs(t, record, 0, 24, 47, 52)

	workout := resolver.FilterByCategory("WORKOUT")
	assertTypeIDs(t, workout, 26)
}

func TestAllTypes(t *testing.T) {
	resolver := NewTypeResolver(testHealthTypesResponse())

	allTypes := resolver.AllTypes()
	assertTypeIDs(t, allTypes, 5, 6, 9, 0, 24, 47, 52, 26)

	allTypes[0].Name = "mutated"
	fresh := resolver.AllTypes()
	if fresh[0].Name != "Distance cycling" {
		t.Fatalf("AllTypes() returned shared slice, got %q after mutation", fresh[0].Name)
	}
}

func testHealthTypesResponse() *api.HealthTypesResponse {
	return &api.HealthTypesResponse{
		Aggregated: []api.HealthTypeSection{
			{
				Name: "Activity",
				Types: []api.HealthType{
					{ID: 5, Name: "Distance cycling", Category: "Cumulative", SubCategory: "Activity"},
					{ID: 6, Name: "Heart rate", Category: "Cumulative", SubCategory: "Heart rate"},
					{ID: 9, Name: "Step count", Category: "Cumulative", SubCategory: "Activity"},
				},
			},
		},
		Record: []api.HealthTypeSection{
			{
				Name: "Body",
				Types: []api.HealthType{
					{ID: 0, Name: "Body mass", Category: "Record", SubCategory: "Body"},
					{ID: 24, Name: "Time in bed", Category: "Record", SubCategory: "Sleep"},
					{ID: 47, Name: "Body mass index", Category: "Record", SubCategory: "Body"},
				},
			},
			{
				Name: "Heart rate",
				Types: []api.HealthType{
					{ID: 52, Name: "Heart rate", Category: "Record", SubCategory: "Heart rate"},
				},
			},
		},
		Workout: []api.HealthTypeSection{
			{
				Name: "Workout",
				Types: []api.HealthType{
					{ID: 26, Name: "Workouts", Category: "Workout", SubCategory: "Workout"},
				},
			},
		},
	}
}

func assertTypeIDs(t *testing.T, got []api.HealthType, want ...int) {
	t.Helper()

	gotIDs := make([]int, 0, len(got))
	for _, healthType := range got {
		gotIDs = append(gotIDs, healthType.ID)
	}

	if !slices.Equal(gotIDs, want) {
		t.Fatalf("type IDs = %v, want %v", gotIDs, want)
	}
}
