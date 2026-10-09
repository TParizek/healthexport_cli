package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/TParizek/healthexport_cli/internal/service"
)

func FormatWorkoutList(w io.Writer, format string, workouts []service.WorkoutSummary) error {
	if format == "json" {
		return writeJSON(w, nonNilSlice(workouts))
	}
	if format != "csv" {
		return fmt.Errorf("unsupported format %q", format)
	}
	writer := csv.NewWriter(w)
	headers := []string{"workout_id", "revision", "start", "end", "sport", "source", "active_duration_seconds", "elapsed_duration_seconds", "distance_meters"}
	for _, s := range service.WorkoutSections {
		headers = append(headers, "available_"+snakeSection(s))
	}
	if err := writer.Write(headers); err != nil {
		return err
	}
	for _, workout := range workouts {
		m := workout.Manifest
		distance := ""
		if m.TotalDistance != nil {
			distance = strconv.FormatFloat(*m.TotalDistance, 'f', -1, 64)
		}
		row := []string{workout.WorkoutID, workout.Revision, m.Start, m.End, m.Sport, m.Source, scalar(m.ActiveDuration), scalar(m.ElapsedDuration), distance}
		for _, section := range service.WorkoutSections {
			status := m.Availability[section]
			if status == "" {
				status = "unavailable"
			}
			row = append(row, status)
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
func snakeSection(s string) string {
	switch s {
	case "heartRate":
		return "heart_rate"
	case "splitsKm":
		return "splits_km"
	case "splitsMi":
		return "splits_mi"
	}
	return s
}
func scalar(value any) string {
	if value == nil {
		return ""
	}
	if number, ok := value.(float64); ok {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return fmt.Sprint(value)
}
func FormatWorkoutSection(w io.Writer, format string, section service.WorkoutSection) error {
	if format == "json" {
		return writeJSON(w, section)
	}
	if format != "csv" {
		return fmt.Errorf("unsupported format %q", format)
	}
	fields := []string{"t", "end", "value", "kind", "min", "max", "count", "coverage", "breakBefore"}
	names := []string{"t_seconds", "end_seconds", "value", "kind", "min", "max", "count", "coverage_seconds", "break_before"}
	if section.Section == "events" {
		fields = []string{"kind", "start", "end"}
		names = []string{"kind", "start_seconds", "end_seconds"}
	}
	if section.Section == "splitsKm" || section.Section == "splitsMi" {
		fields = []string{"index", "distance", "start", "end", "elapsedDuration", "activeDuration", "averageHeartRate", "estimated"}
		names = []string{"index", "distance_meters", "start_seconds", "end_seconds", "elapsed_duration_seconds", "active_duration_seconds", "average_heart_rate_bpm", "estimated"}
	}
	headers := append([]string{"workout_id", "revision", "section", "units", "availability", "aggregation_interval_seconds", "aggregation_method", "aggregation_source"}, names...)
	writer := csv.NewWriter(w)
	if err := writer.Write(headers); err != nil {
		return err
	}
	for _, item := range section.Items {
		interval, method, source := "", "", ""
		if a := section.Aggregation; a != nil {
			interval, method, source = strconv.Itoa(a.IntervalSeconds), a.Method, a.Source
		}
		row := []string{section.WorkoutID, section.Revision, section.Section, section.Unit, section.Availability, interval, method, source}
		for _, field := range fields {
			row = append(row, scalar(item[field]))
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
func FormatWorkoutExport(w io.Writer, data service.WorkoutExport) error { return writeJSON(w, data) }
