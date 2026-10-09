package output

import (
	"bytes"
	"encoding/csv"
	"errors"
	"strings"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/service"
)

type brokenWorkoutWriter struct{}

func (brokenWorkoutWriter) Write([]byte) (int, error) { return 0, errors.New("writer failed") }
func TestWorkoutCSVAbsentZeroAndFalse(t *testing.T) {
	var out bytes.Buffer
	data := service.WorkoutSection{Section: "speed", Unit: "m/s", Items: []map[string]any{{"t": 0.0, "value": 0.0, "kind": "sample", "breakBefore": false}}}
	if err := FormatWorkoutSection(&out, "csv", data); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(out.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for i, key := range rows[0] {
		values[key] = rows[1][i]
	}
	if values["value"] != "0" || values["break_before"] != "false" || values["min"] != "" || values["end_seconds"] != "" {
		t.Fatal(values)
	}
	if err := FormatWorkoutSection(brokenWorkoutWriter{}, "csv", data); err == nil {
		t.Fatal("ignored write failure")
	}
	if err := FormatWorkoutList(brokenWorkoutWriter{}, "csv", nil); err == nil {
		t.Fatal("ignored write failure")
	}
}
