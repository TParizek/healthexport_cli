package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/workouttest"
)

func fixtureReader(t *testing.T, mutate func(*http.Request, map[string]any)) *WorkoutReader {
	t.Helper()
	server := httptest.NewServer(workouttest.Handler(mutate))
	t.Cleanup(server.Close)
	reader, err := NewWorkoutReader(Options{APIURL: server.URL + "/api/v2", AccountKey: workouttest.Account})
	if err != nil {
		t.Fatal(err)
	}
	return reader
}
func TestCompleteWorkoutExportAndScopedReads(t *testing.T) {
	sections := []string{}
	reader := fixtureReader(t, func(r *http.Request, _ map[string]any) { sections = append(sections, r.URL.Query().Get("section")) })
	q := WorkoutPageQuery{WorkoutID: workouttest.ID, Revision: workouttest.Revision, Section: "speed", Unit: "km/h"}
	series, err := reader.FetchWorkoutSection(context.Background(), q)
	if err != nil || len(series.Items) != 9 || len(sections) != 2 || sections[0] != "speed" || sections[1] != "speed" || series.Unit != "km/h" {
		t.Fatalf("scoped complete series: %#v %v %v", series, sections, err)
	}
	if series.Items[8]["breakBefore"] != true || series.Items[0]["value"] != float64(0) {
		t.Fatal("lost zero/gap")
	}
	exported, err := reader.ExportWorkout(context.Background(), q.WorkoutID, q.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(exported.Sections["speed"]) != 9 || len(exported.Sections["splitsKm"]) != 1 || len(exported.Sections["splitsMi"]) != 1 || exported.Sections["heartRate"] == nil {
		t.Fatal("incomplete export")
	}
	if exported.Sections["splitsMi"][0]["distance"] != 1609.344 || exported.Units["distance"] != "meters" {
		t.Fatal("noncanonical export")
	}
	if _, ok := exported.Sections["splitsKm"][0]["averageHeartRate"]; ok {
		t.Fatal("fabricated missing value")
	}
	// Manifest survives the JSON boundary without adding fields or changing values.
	got, _ := json.Marshal(exported.Manifest)
	want, _ := json.Marshal(workouttest.Manifest())
	var a, b any
	_ = json.Unmarshal(got, &a)
	_ = json.Unmarshal(want, &b)
	ga, _ := json.Marshal(a)
	gb, _ := json.Marshal(b)
	if string(ga) != string(gb) {
		t.Fatalf("manifest mismatch\n%s\n%s", ga, gb)
	}
}
func TestCompleteWorkoutRejectsIncompleteAndChangedPages(t *testing.T) {
	for _, mode := range []string{"early-end", "missing-page", "repeated-index", "wrong-next", "changed-manifest", "tampered", "wrong-revision", "empty-stored-page"} {
		t.Run(mode, func(t *testing.T) {
			reader := fixtureReader(t, func(r *http.Request, body map[string]any) {
				if r.URL.Query().Get("section") != "speed" {
					return
				}
				switch mode {
				case "empty-stored-page":
					p := body["pages"].([]any)[0].(map[string]any)
					p["sealed"] = workouttest.Seal("0", map[string]any{"section": "speed", "items": []any{}})
				case "early-end":
					body["nextOffset"] = nil
				case "missing-page":
					body["pages"] = []any{}
				case "wrong-next":
					body["nextOffset"] = 99
				case "wrong-revision":
					body["revision"] = "33333333-3333-4333-8333-333333333333"
				case "tampered":
					body["manifest"] = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
				case "changed-manifest":
					if r.URL.Query().Get("offset") == "8" {
						m := workouttest.Manifest()
						m["source"] = "Changed"
						body["manifest"] = workouttest.Seal("manifest", m)
					}
				case "repeated-index":
					if r.URL.Query().Get("offset") == "8" {
						page := body["pages"].([]any)[0].(map[string]any)
						page["index"] = 0
						page["sealed"] = workouttest.Seal("0", map[string]any{"section": "speed", "items": []any{map[string]any{"t": 80, "value": 2, "kind": "sample"}}})
					}
				}
			})
			result, err := reader.ExportWorkout(context.Background(), workouttest.ID, workouttest.Revision)
			if err == nil || result.Sections != nil {
				t.Fatalf("accepted %s", mode)
			}
		})
	}
}
func TestWorkoutListDatesPaginationAndBounds(t *testing.T) {
	calls := 0
	reader := fixtureReader(t, func(r *http.Request, body map[string]any) {
		calls++
		if r.URL.Query().Get("dateFrom") != "2026-10-01T00:00:00.000Z" || r.URL.Query().Get("dateTo") != "2026-10-02T00:00:00.000Z" {
			t.Error("wrong date semantics")
		}
		if calls == 1 {
			body["nextOffset"] = 1
		} else {
			body["workouts"] = []any{}
		}
	})
	workouts, err := reader.ListAllWorkouts(context.Background(), "2026-10-02", "2026-10-01")
	if err != nil || len(workouts) != 1 || calls != 2 {
		t.Fatalf("%v %v %d", workouts, err, calls)
	}
	for _, dates := range [][2]string{{"2025-01-01", "2026-10-02"}, {"bad", "2026-10-02"}, {"2026-10-01T00:00:00.000001Z", "2026-10-02"}} {
		_, err := reader.ListAllWorkouts(context.Background(), dates[0], dates[1])
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid dates accepted", dates, err)
		}
	}
	if calls != 2 {
		t.Fatal("invalid query reached backend")
	}
}
func TestWorkoutRevisionConflictRetainsAPIError(t *testing.T) {
	reader := fixtureReader(t, nil)
	_, err := reader.ExportWorkout(context.Background(), workouttest.ID, "33333333-3333-4333-8333-333333333333")
	var apiErr *api.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 409 {
		t.Fatal(err)
	}
}
func TestUnavailableWorkoutSectionIsNotZero(t *testing.T) {
	reader := fixtureReader(t, nil)
	result, err := reader.FetchWorkoutSection(context.Background(), WorkoutPageQuery{WorkoutID: workouttest.ID, Revision: workouttest.Revision, Section: "heartRate"})
	if err != nil || result.Availability != "unavailable" || result.Items == nil || len(result.Items) != 0 {
		t.Fatal(result, err)
	}
}

// Golden produced by HE-FE-Remote createWorkoutDetailJsonExport with the
// same synthetic fixture (not a Go reimplementation of the expected output).
func TestWorkoutExportMatchesWebsiteGolden(t *testing.T) {
	reader := fixtureReader(t, nil)
	got, err := reader.ExportWorkout(context.Background(), workouttest.ID, workouttest.Revision)
	if err != nil {
		t.Fatal(err)
	}
	golden, err := os.ReadFile("testdata/workout-web-export.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(golden, &b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("export differs from website fixture")
	}
}
