package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/service"
	"github.com/TParizek/healthexport_cli/internal/workouttest"
)

func TestWorkoutStdioToolsAndPagination(t *testing.T) {
	backend := httptest.NewServer(workouttest.Handler(nil))
	defer backend.Close()
	requests := []map[string]any{
		{"jsonrpc": "2.0", "id": 1, "method": "tools/list"},
	}
	calls := []struct {
		name string
		args map[string]any
	}{
		{"list_workouts", map[string]any{"start": "2026-10-01T00:00:00Z", "end": "2026-10-02T00:00:00Z"}},
		{"fetch_workout_series", map[string]any{"workoutId": workouttest.ID, "revision": workouttest.Revision, "section": "speed", "limit": 8}},
		{"fetch_workout_series", map[string]any{"workoutId": workouttest.ID, "revision": workouttest.Revision, "section": "speed", "offset": 8}},
		{"fetch_workout_detail", map[string]any{"workoutId": workouttest.ID, "revision": workouttest.Revision, "splitUnit": "mi"}},
	}
	for i, call := range calls {
		requests = append(requests, map[string]any{"jsonrpc": "2.0", "id": i + 2, "method": "tools/call", "params": map[string]any{"name": call.name, "arguments": call.args}})
	}
	var in, out bytes.Buffer
	for _, r := range requests {
		if err := json.NewEncoder(&in).Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	server := NewServer(service.Options{AccountKey: workouttest.Account, APIURL: backend.URL + "/api/v2"}, "test", "test", &in, &out)
	if err := server.Serve(context.Background()); err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(&out)
	for i := range requests {
		var response map[string]any
		if err := decoder.Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response["error"] != nil {
			t.Fatal(response)
		}
		result := response["result"].(map[string]any)
		if i == 0 {
			names := map[string]bool{}
			for _, v := range result["tools"].([]any) {
				names[v.(map[string]any)["name"].(string)] = true
			}
			for _, name := range []string{"list_workouts", "fetch_workout_detail", "fetch_workout_series"} {
				if !names[name] {
					t.Fatal("missing tool", name)
				}
			}
			continue
		}
		structured := result["structuredContent"].(map[string]any)
		var text any
		if err := json.Unmarshal([]byte(result["content"].([]any)[0].(map[string]any)["text"].(string)), &text); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(structured, text) {
			t.Fatal("text and structured content differ")
		}
		switch i {
		case 1:
			if len(structured["workouts"].([]any)) != 1 {
				t.Fatal(structured)
			}
		case 2:
			if len(structured["pages"].([]any)) != 8 || structured["nextOffset"] != float64(8) {
				t.Fatal(structured)
			}
		case 3:
			if len(structured["pages"].([]any)) != 1 || structured["nextOffset"] != nil {
				t.Fatal(structured)
			}
		case 4:
			sections := structured["sections"].([]any)
			if len(sections) != 2 || sections[1].(map[string]any)["section"] != "splitsMi" {
				t.Fatal(structured)
			}
		}
	}
}
func TestWorkoutMCPRejectsUnexpectedAndMalformedArguments(t *testing.T) {
	server := NewServer(service.Options{AccountKey: workouttest.Account}, "test", "test", nil, nil)
	for _, args := range []map[string]any{
		{"start": "2026-10-01T00:00:00Z", "end": "2026-10-02T00:00:00Z", "accountKey": "secret"},
		{"start": "2026-10-01T00:00:00Z", "end": "2026-10-02T00:00:00Z", "limit": 0},
		{"start": "2026-10-01T00:00:00Z", "end": "2026-10-02T00:00:00Z", "limit": 1.5},
		{"start": "2026-10-01T00:00:00Z", "end": "2026-10-02T00:00:00Z", "offset": nil},
		{"start": "2026-10-01T00:00:00Z"},
	} {
		result := server.HandleToolCall("list_workouts", args)
		if !result.IsError {
			t.Fatal("accepted bad args", args)
		}
		if result.StructuredContent.(map[string]any)["error"].(map[string]any)["category"] != "invalid_input" {
			t.Fatal(result)
		}
	}
}
