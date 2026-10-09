package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/TParizek/healthexport_cli/internal/workouttest"
)

func setupWorkoutCommand(t *testing.T, mutate func(*http.Request, map[string]any)) {
	t.Helper()
	oldKey, oldURL := accountKey, apiURL
	t.Cleanup(func() { accountKey, apiURL = oldKey, oldURL })
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	server := httptest.NewServer(workouttest.Handler(mutate))
	t.Cleanup(server.Close)
	accountKey, apiURL = workouttest.Account, server.URL+"/api/v2"
}
func executeWorkouts(args ...string) (string, string, error) {
	cmd := newWorkoutsCommand()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	var out, diagnostics bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&diagnostics)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), diagnostics.String(), err
}
func TestWorkoutCommandsRespectFormatsAndExport(t *testing.T) {
	setupWorkoutCommand(t, nil)
	cfg := &config.Config{Format: "json"}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	out, _, err := executeWorkouts("list", "-f", "2026-10-01", "-T", "2026-10-02")
	if err != nil || !strings.HasPrefix(out, "[") {
		t.Fatal(out, err)
	}
	out, _, err = executeWorkouts("list", "-f", "2026-10-01", "-T", "2026-10-02", "--format", "csv")
	if err != nil || !strings.HasPrefix(out, "workout_id,revision") {
		t.Fatal(out, err)
	}
	cfg.Format = "csv"
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	out, _, err = executeWorkouts("export", workouttest.ID, "--revision", workouttest.Revision)
	var exported map[string]any
	if err != nil || json.Unmarshal([]byte(out), &exported) != nil || exported["exportVersion"] != float64(1) {
		t.Fatal(err, out)
	}
	out, _, err = executeWorkouts("splits", workouttest.ID, "--revision", workouttest.Revision, "--split-unit", "mi")
	if err != nil || !strings.Contains(out, "1609.344") || !strings.Contains(out, "distance_meters") {
		t.Fatal(out, err)
	}
	out, _, err = executeWorkouts("series", workouttest.ID, "--revision", workouttest.Revision, "--section", "speed")
	rows, parseErr := csv.NewReader(strings.NewReader(out)).ReadAll()
	if err != nil || parseErr != nil || len(rows) != 10 || !strings.Contains(out, "break_before") {
		t.Fatal(out, err, parseErr)
	}
	out, diagnostics, err := executeWorkouts("series", workouttest.ID, "--revision", workouttest.Revision, "--section", "heartRate")
	if err != nil || !strings.Contains(diagnostics, "unavailable") || strings.Count(out, "\n") != 1 {
		t.Fatal(out, diagnostics, err)
	}
}
func TestWorkoutCommandLateFailureWritesNoData(t *testing.T) {
	setupWorkoutCommand(t, func(r *http.Request, body map[string]any) {
		if r.URL.Query().Get("offset") == "8" {
			body["manifest"] = "tampered"
		}
	})
	for _, args := range [][]string{{"export", workouttest.ID, "--revision", workouttest.Revision}, {"series", workouttest.ID, "--revision", workouttest.Revision, "--section", "speed"}} {
		out, _, err := executeWorkouts(args...)
		if err == nil || out != "" {
			t.Fatal("partial output", len(out), err)
		}
	}
}
func TestWorkoutCommandArgumentsAndConflict(t *testing.T) {
	setupWorkoutCommand(t, nil)
	for _, args := range [][]string{{"export"}, {"export", workouttest.ID}, {"export", workouttest.ID, "--revision", workouttest.Revision, "--format", "csv"}, {"series", workouttest.ID, "--revision", workouttest.Revision}, {"series", workouttest.ID, "--revision", workouttest.Revision, "--section", "speed", "--unit", "bpm"}, {"splits", workouttest.ID, "--revision", workouttest.Revision, "--split-unit", "feet"}} {
		out, _, err := executeWorkouts(args...)
		if err == nil || out != "" {
			t.Fatal(args, out, err)
		}
	}
	out, _, err := executeWorkouts("events", workouttest.ID, "--revision", "33333333-3333-4333-8333-333333333333")
	if out != "" || exitCodeForError(err) != 3 || !strings.Contains(err.Error(), "list again") {
		t.Fatal(out, err)
	}
}
