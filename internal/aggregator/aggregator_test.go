package aggregator

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestAggregateDay(t *testing.T) {
	records := []DecryptedRecord{
		{Time: "2024-01-01T08:00:00Z", Value: "10"},
		{Time: "2024-01-01T18:00:00Z", Value: "5.5"},
		{Time: "2024-01-02T08:00:00Z", Value: "3"},
		{Time: "2024-01-03T08:00:00Z", Value: "-1"},
		{Time: "2024-01-03T09:00:00Z", Value: "2"},
	}

	got, skipped, err := Aggregate(records, PeriodDay)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	want := []AggregatedRecord{
		{Period: "2024-01-01", Value: 15.5},
		{Period: "2024-01-02", Value: 3},
		{Period: "2024-01-03", Value: 1},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateWeek(t *testing.T) {
	records := []DecryptedRecord{
		{Time: "2024-01-01T09:00:00Z", Value: "1"},
		{Time: "2024-01-07T09:00:00Z", Value: "2"},
		{Time: "2024-01-08T09:00:00Z", Value: "3"},
		{Time: "2024-01-15T09:00:00Z", Value: "4"},
	}

	got, skipped, err := Aggregate(records, PeriodWeek)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	want := []AggregatedRecord{
		{Period: "2024-W01", Value: 3},
		{Period: "2024-W02", Value: 3},
		{Period: "2024-W03", Value: 4},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateMonth(t *testing.T) {
	records := []DecryptedRecord{
		{Time: "2024-01-10T09:00:00Z", Value: "1"},
		{Time: "2024-02-10T09:00:00Z", Value: "2"},
		{Time: "2024-02-11T09:00:00Z", Value: "3"},
		{Time: "2024-03-10T09:00:00Z", Value: "4"},
	}

	got, skipped, err := Aggregate(records, PeriodMonth)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	want := []AggregatedRecord{
		{Period: "2024-01", Value: 1},
		{Period: "2024-02", Value: 5},
		{Period: "2024-03", Value: 4},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateYear(t *testing.T) {
	records := []DecryptedRecord{
		{Time: "2023-12-10T09:00:00Z", Value: "1"},
		{Time: "2024-01-10T09:00:00Z", Value: "2"},
		{Time: "2024-12-10T09:00:00Z", Value: "3"},
	}

	got, skipped, err := Aggregate(records, PeriodYear)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	want := []AggregatedRecord{
		{Period: "2023", Value: 1},
		{Period: "2024", Value: 5},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateSingleRecord(t *testing.T) {
	got, skipped, err := Aggregate([]DecryptedRecord{
		{Time: "2024-01-15T12:00:00Z", Value: "42"},
	}, PeriodDay)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	want := []AggregatedRecord{{Period: "2024-01-15", Value: 42}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateEmptyInput(t *testing.T) {
	got, skipped, err := Aggregate(nil, PeriodDay)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	if len(got) != 0 {
		t.Fatalf("len(Aggregate()) = %d, want 0", len(got))
	}
}

func TestAggregateSkipsNonNumericValues(t *testing.T) {
	records := []DecryptedRecord{
		{Time: "2024-01-01T08:00:00Z", Value: "10"},
		{Time: "2024-01-01T09:00:00Z", Value: "abc"},
		{Time: "2024-01-02T09:00:00Z", Value: " 3.5 "},
		{Time: "2024-01-02T10:00:00Z", Value: ""},
	}

	got, skipped, err := Aggregate(records, PeriodDay)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 2 {
		t.Fatalf("skipped = %d, want 2", skipped)
	}

	want := []AggregatedRecord{
		{Period: "2024-01-01", Value: 10},
		{Period: "2024-01-02", Value: 3.5},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateAllNonNumeric(t *testing.T) {
	records := []DecryptedRecord{
		{Time: "2024-01-01T08:00:00Z", Value: "nope"},
		{Time: "2024-01-02T08:00:00Z", Value: ""},
	}

	got, skipped, err := Aggregate(records, PeriodDay)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != len(records) {
		t.Fatalf("skipped = %d, want %d", skipped, len(records))
	}

	if len(got) != 0 {
		t.Fatalf("len(Aggregate()) = %d, want 0", len(got))
	}
}

func TestAggregateChronologicalOrdering(t *testing.T) {
	records := []DecryptedRecord{
		{Time: "2024-01-03T08:00:00Z", Value: "3"},
		{Time: "2024-01-01T08:00:00Z", Value: "1"},
		{Time: "2024-01-02T08:00:00Z", Value: "2"},
	}

	got, _, err := Aggregate(records, PeriodDay)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	wantPeriods := []string{"2024-01-01", "2024-01-02", "2024-01-03"}
	for i, want := range wantPeriods {
		if got[i].Period != want {
			t.Fatalf("got[%d].Period = %q, want %q", i, got[i].Period, want)
		}
	}
}

func TestAggregateFloatingPointPrecision(t *testing.T) {
	got, skipped, err := Aggregate([]DecryptedRecord{
		{Time: "2024-01-01T08:00:00Z", Value: "0.1"},
		{Time: "2024-01-01T09:00:00Z", Value: "0.2"},
	}, PeriodDay)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	if diff := math.Abs(got[0].Value - 0.3); diff > 1e-9 {
		t.Fatalf("got[0].Value = %.17f, want close to 0.3", got[0].Value)
	}
}

func TestAggregateIncludesZeroValues(t *testing.T) {
	got, skipped, err := Aggregate([]DecryptedRecord{
		{Time: "2024-01-01T08:00:00Z", Value: "0"},
		{Time: "2024-01-01T09:00:00Z", Value: "5"},
		{Time: "2024-01-02T09:00:00Z", Value: "0"},
	}, PeriodDay)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	want := []AggregatedRecord{
		{Period: "2024-01-01", Value: 5},
		{Period: "2024-01-02", Value: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateWeekBoundary(t *testing.T) {
	got, skipped, err := Aggregate([]DecryptedRecord{
		{Time: "2024-01-07T12:00:00Z", Value: "1"},
		{Time: "2024-01-08T12:00:00Z", Value: "2"},
	}, PeriodWeek)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	want := []AggregatedRecord{
		{Period: "2024-W01", Value: 1},
		{Period: "2024-W02", Value: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateYearBoundary(t *testing.T) {
	got, skipped, err := Aggregate([]DecryptedRecord{
		{Time: "2024-12-31T12:00:00Z", Value: "1"},
		{Time: "2025-01-01T12:00:00Z", Value: "2"},
	}, PeriodYear)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if skipped != 0 {
		t.Fatalf("skipped = %d, want 0", skipped)
	}

	want := []AggregatedRecord{
		{Period: "2024", Value: 1},
		{Period: "2025", Value: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aggregate() = %#v, want %#v", got, want)
	}
}

func TestAggregateInvalidTime(t *testing.T) {
	_, _, err := Aggregate([]DecryptedRecord{
		{Time: "not-a-time", Value: "1"},
	}, PeriodDay)
	if err == nil {
		t.Fatal("Aggregate() error = nil, want error")
	}
}

func TestAggregateInvalidPeriod(t *testing.T) {
	_, _, err := Aggregate([]DecryptedRecord{
		{Time: "2024-01-01T08:00:00Z", Value: "1"},
	}, Period("quarter"))
	if err == nil {
		t.Fatal("Aggregate() error = nil, want error")
	}
}

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		input   string
		want    Period
		wantErr bool
	}{
		{input: "day", want: PeriodDay},
		{input: "week", want: PeriodWeek},
		{input: "month", want: PeriodMonth},
		{input: "year", want: PeriodYear},
		{input: " DAY ", want: PeriodDay},
		{input: "quarter", wantErr: true},
	}

	for _, tt := range tests {
		got, err := ParsePeriod(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("ParsePeriod(%q) error = nil, want error", tt.input)
			}
			continue
		}

		if err != nil {
			t.Fatalf("ParsePeriod(%q) error = %v", tt.input, err)
		}

		if got != tt.want {
			t.Fatalf("ParsePeriod(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidPeriods(t *testing.T) {
	want := []string{"day", "week", "month", "year"}
	if got := ValidPeriods(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ValidPeriods() = %#v, want %#v", got, want)
	}
}

func TestBucketLabel(t *testing.T) {
	ts := time.Date(2024, time.January, 1, 0, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))

	tests := []struct {
		period Period
		want   string
	}{
		{period: PeriodDay, want: "2023-12-31"},
		{period: PeriodWeek, want: "2023-W52"},
		{period: PeriodMonth, want: "2023-12"},
		{period: PeriodYear, want: "2023"},
	}

	for _, tt := range tests {
		if got := BucketLabel(ts, tt.period); got != tt.want {
			t.Fatalf("BucketLabel(%q) = %q, want %q", tt.period, got, tt.want)
		}
	}
}
