package aggregator

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TParizek/healthexport_cli/internal/api"
)

type (
	DecryptedRecord  = api.DecryptedRecord
	AggregatedRecord = api.AggregatedRecord
)

type Period string

var ErrNotAggregatable = errors.New("type is not aggregatable")

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
	PeriodYear  Period = "year"
)

var validPeriods = []string{
	string(PeriodDay),
	string(PeriodWeek),
	string(PeriodMonth),
	string(PeriodYear),
}

func ParsePeriod(s string) (Period, error) {
	period := Period(strings.ToLower(strings.TrimSpace(s)))
	if !isValidPeriod(period) {
		return "", fmt.Errorf("invalid aggregate period %q (valid: %s)", s, strings.Join(validPeriods, ", "))
	}

	return period, nil
}

func ValidPeriods() []string {
	return slices.Clone(validPeriods)
}

func Aggregate(records []DecryptedRecord, period Period) ([]AggregatedRecord, int, error) {
	if !isValidPeriod(period) {
		return nil, 0, fmt.Errorf("invalid aggregate period %q", period)
	}

	sums := make(map[string]float64)
	skipped := 0

	for _, record := range records {
		timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(record.Time))
		if err != nil {
			return nil, skipped, fmt.Errorf("parse record time %q: %w", record.Time, err)
		}

		value, err := strconv.ParseFloat(strings.TrimSpace(record.Value), 64)
		if err != nil {
			skipped++
			continue
		}

		label := BucketLabel(timestamp, period)
		sums[label] += value
	}

	labels := make([]string, 0, len(sums))
	for label := range sums {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	aggregated := make([]AggregatedRecord, 0, len(labels))
	for _, label := range labels {
		aggregated = append(aggregated, AggregatedRecord{
			Period: label,
			Value:  sums[label],
		})
	}

	return aggregated, skipped, nil
}

func BucketLabel(t time.Time, period Period) string {
	utc := t.UTC()

	switch period {
	case PeriodDay:
		return utc.Format("2006-01-02")
	case PeriodWeek:
		year, week := utc.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	case PeriodMonth:
		return utc.Format("2006-01")
	case PeriodYear:
		return utc.Format("2006")
	default:
		return ""
	}
}

func isValidPeriod(period Period) bool {
	switch period {
	case PeriodDay, PeriodWeek, PeriodMonth, PeriodYear:
		return true
	default:
		return false
	}
}
