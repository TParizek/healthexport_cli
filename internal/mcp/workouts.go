package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/TParizek/healthexport_cli/internal/service"
)

func workoutToolDefinitions() []toolDefinition {
	str := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	integer := func(minimum, maximum, defaultValue int) map[string]any {
		return map[string]any{"type": "integer", "minimum": minimum, "maximum": maximum, "default": defaultValue}
	}
	definition := func(name, description string, properties map[string]any, required []string) toolDefinition {
		return toolDefinition{Name: name, Description: description, InputSchema: map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}, OutputSchema: map[string]any{"type": "object"}, Annotations: map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": true}}
	}
	identity := func() map[string]any {
		return map[string]any{"workoutId": str("Workout ID from list_workouts"), "revision": str("Revision from list_workouts; refresh listing on conflict"), "offset": integer(0, 512, 0)}
	}
	list := definition("list_workouts", "List uploaded enriched workout manifests, no sample streams. At most 366 days and 10 workouts per page; use nextOffset. Old summary-only workouts are not included. Does not trigger sync.", map[string]any{"start": str("Inclusive RFC3339 start with timezone, at most millisecond precision"), "end": str("Inclusive RFC3339 end, no more than 366 days after start"), "offset": integer(0, 10000, 0), "limit": integer(1, 10, 10)}, []string{"start", "end"})
	detailProps := identity()
	detailProps["limit"] = integer(1, 4, 1)
	detailProps["splitUnit"] = map[string]any{"type": "string", "enum": []string{"km", "mi"}, "default": "km", "description": "Select recorded splits; their distance values remain meters"}
	detail := definition("fetch_workout_detail", "Read one workout manifest and bounded events/splits pages, without numeric streams. Each section has its own nextOffset; repeat with that offset. Times seconds, distances meters. Does not trigger sync.", detailProps, []string{"workoutId", "revision"})
	seriesProps := identity()
	seriesProps["limit"] = integer(1, 8, 1)
	seriesProps["section"] = map[string]any{"type": "string", "enum": []string{"heartRate", "distance", "speed", "elevation", "power", "cadence"}}
	seriesProps["unit"] = str("Optional canonical or compatible converted unit: bpm, m, m/s, W, rpm, km, mi, ft, km/h, mph")
	series := definition("fetch_workout_series", "Read one explicitly selected numeric section for one workout/revision, 1–8 pages of at most 256 stored points each. Follow nextOffset. Includes aggregation interval, method and source. Values and min/max use returned unit; t/end/coverage are seconds. breakBefore marks gaps. Stored means cannot recover raw peaks or exact effort durations. Does not trigger sync.", seriesProps, []string{"workoutId", "revision", "section"})
	return []toolDefinition{list, detail, series}
}

func decodeWorkoutArguments(arguments map[string]any, target any, required ...string) error {
	for _, key := range required {
		value, ok := arguments[key].(string)
		if !ok || value == "" {
			return fmt.Errorf("%w: %s is required and must be a string", service.ErrInvalidInput, key)
		}
	}
	// Reject null values, including optional fields: omission uses the default.
	for key, value := range arguments {
		if value == nil {
			return fmt.Errorf("%w: %s must not be null", service.ErrInvalidInput, key)
		}
	}
	raw, err := json.Marshal(arguments)
	if err != nil {
		return fmt.Errorf("%w: invalid arguments", service.ErrInvalidInput)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil {
		return fmt.Errorf("%w: unexpected argument or invalid argument type", service.ErrInvalidInput)
	}
	if value, ok := arguments["limit"]; ok {
		if n, ok := value.(float64); ok && n == 0 {
			return fmt.Errorf("%w: limit must be positive", service.ErrInvalidInput)
		}
		if n, ok := value.(int); ok && n == 0 {
			return fmt.Errorf("%w: limit must be positive", service.ErrInvalidInput)
		}
	}
	return nil
}
func (s *Server) handleWorkoutTool(name string, arguments map[string]any) toolResult {
	var list service.WorkoutListQuery
	var detail service.WorkoutDetailQuery
	var series service.WorkoutPageQuery
	var err error
	switch name {
	case "list_workouts":
		err = decodeWorkoutArguments(arguments, &list, "start", "end")
	case "fetch_workout_detail":
		err = decodeWorkoutArguments(arguments, &detail, "workoutId", "revision")
	case "fetch_workout_series":
		err = decodeWorkoutArguments(arguments, &series, "workoutId", "revision", "section")
	}
	if err != nil {
		return s.mapToolError(err)
	}
	reader, err := service.NewWorkoutReader(s.options)
	if err != nil {
		return s.mapToolError(err)
	}
	var result any
	switch name {
	case "list_workouts":
		result, err = reader.ListWorkouts(context.Background(), list)
	case "fetch_workout_detail":
		result, err = reader.GetWorkoutDetail(context.Background(), detail)
	case "fetch_workout_series":
		result, err = reader.GetWorkoutSeries(context.Background(), series)
	}
	if err != nil {
		return s.mapToolError(err)
	}
	return s.toolSuccess(result)
}
