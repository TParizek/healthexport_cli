package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/TParizek/healthexport_cli/internal/service"
)

const testAccountKey = "abcdef.0123456789abcdef0123456789abcdef.gh01"

func TestHandleToolCallRejectsUnexpectedArgument(t *testing.T) {
	server := NewServer(service.Options{}, "1.0.0", "1.0.0", bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	result := server.HandleToolCall("list_health_types", map[string]any{"unexpected": true})
	if !result.IsError {
		t.Fatal("IsError = false, want true")
	}

	if got := result.StructuredContent.(map[string]any)["error"].(map[string]any)["category"]; got != "invalid_input" {
		t.Fatalf("category = %v, want invalid_input", got)
	}
}

func TestHandleToolCallListHealthTypesMatchesDeclaredSchemaShape(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeJSONResponse(t, w, &api.HealthTypesResponse{
				Aggregated: []api.HealthTypeSection{
					{
						Name: "Activity",
						Types: []api.HealthType{
							{ID: 9, Name: "Step count", Category: "Cumulative", SubCategory: "Activity"},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiServer.Close)

	server := NewServer(service.Options{APIURL: apiServer.URL}, "1.0.0", "1.0.0", bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	result := server.HandleToolCall("list_health_types", map[string]any{"category": "aggregated"})
	if result.IsError {
		t.Fatalf("IsError = true, want false with payload %#v", result.StructuredContent)
	}

	payload := result.StructuredContent.(map[string]any)
	types, ok := payload["types"].([]mcpHealthType)
	if !ok {
		t.Fatalf("types type = %T, want []mcpHealthType", payload["types"])
	}

	if len(types) != 1 {
		t.Fatalf("len(types) = %d, want 1", len(types))
	}

	if got, want := types[0].Subcategory, "Activity"; got != want {
		t.Fatalf("Subcategory = %q, want %q", got, want)
	}

	body := result.Content[0].Text
	if !strings.Contains(body, "\"subcategory\":\"Activity\"") {
		t.Fatalf("content = %q, want subcategory field", body)
	}

	if strings.Contains(body, "\"subCategory\"") {
		t.Fatalf("content = %q, want no subCategory field", body)
	}
}

func TestHandleToolCallFetchHealthDataReturnsStructuredContent(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	if err := (&config.Config{AccountKey: testAccountKey}).SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeJSONResponse(t, w, &api.HealthTypesResponse{
				Aggregated: []api.HealthTypeSection{
					{
						Name: "Activity",
						Types: []api.HealthType{
							{ID: 9, Name: "Step count", Category: "Cumulative", SubCategory: "Activity"},
						},
					},
				},
			})
		case "/healthdata/encrypted":
			writeJSONResponse(t, w, []api.EncryptedPackage{
				{
					Type: 9,
					Data: []api.EncryptedUnitGroup{
						{
							Units: "count",
							Records: []api.EncryptedRecord{
								{
									Time:   "2024-01-14T12:00:00Z",
									Nonce:  "AQIDBAUGBwgJCgsM",
									Cipher: "lmaYhg==",
								},
							},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiServer.Close)

	server := NewServer(service.Options{
		ConfigPath: configPath,
		APIURL:     apiServer.URL,
	}, "1.0.0", "1.0.0", bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	result := server.HandleToolCall("fetch_health_data", map[string]any{
		"types": []any{"step_count"},
		"from":  "2024-01-14",
		"to":    "2024-01-15",
	})
	if result.IsError {
		t.Fatalf("IsError = true, want false with payload %#v", result.StructuredContent)
	}

	payload := result.StructuredContent.(map[string]any)
	if got, want := payload["from"], "2024-01-14T00:00:00Z"; got != want {
		t.Fatalf("from = %v, want %v", got, want)
	}

	results, ok := payload["results"].([]api.DecryptedPackage)
	if !ok {
		t.Fatalf("results type = %T, want []api.DecryptedPackage", payload["results"])
	}

	if got, want := results[0].TypeName, "Step count"; got != want {
		t.Fatalf("TypeName = %q, want %q", got, want)
	}
}

func TestHandleRequestReturnsMCPErrorForInvalidFetchHealthDataInput(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	if err := (&config.Config{AccountKey: testAccountKey}).SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeJSONResponse(t, w, &api.HealthTypesResponse{
				Aggregated: []api.HealthTypeSection{
					{
						Name: "Activity",
						Types: []api.HealthType{
							{ID: 9, Name: "Step count", Category: "Cumulative", SubCategory: "Activity"},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiServer.Close)

	server := NewServer(service.Options{
		ConfigPath: configPath,
		APIURL:     apiServer.URL,
	}, "1.0.0", "1.0.0", bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	cases := []struct {
		name        string
		arguments   map[string]any
		wantMessage string
	}{
		{
			name: "empty types",
			arguments: map[string]any{
				"types": []any{},
				"from":  "2026-03-20",
				"to":    "2026-03-26",
			},
			wantMessage: "the 'types' parameter must contain at least one health type",
		},
		{
			name: "unknown name with aggregate",
			arguments: map[string]any{
				"types":     []any{"nonexistent_type"},
				"from":      "2026-03-20",
				"to":        "2026-03-26",
				"aggregate": "day",
			},
			wantMessage: "unknown health type: nonexistent_type",
		},
		{
			name: "unknown name without aggregate",
			arguments: map[string]any{
				"types": []any{"nonexistent_type"},
				"from":  "2026-03-20",
				"to":    "2026-03-26",
			},
			wantMessage: "unknown health type: nonexistent_type",
		},
		{
			name: "unknown numeric id",
			arguments: map[string]any{
				"types":     []any{999.0},
				"from":      "2026-03-20",
				"to":        "2026-03-26",
				"aggregate": "day",
			},
			wantMessage: "unknown health type ID: 999",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, shouldRespond := server.handleRequest(request{
				JSONRPC: "2.0",
				ID:      json.RawMessage("1"),
				Method:  "tools/call",
				Params: mustJSONRaw(t, map[string]any{
					"name":      "fetch_health_data",
					"arguments": tc.arguments,
				}),
			})
			if !shouldRespond {
				t.Fatal("shouldRespond = false, want true")
			}

			if resp.Error == nil {
				t.Fatalf("Error = nil, want MCP error response; result = %#v", resp.Result)
			}

			if got, want := resp.Error.Code, -32602; got != want {
				t.Fatalf("Error.Code = %d, want %d", got, want)
			}

			if !strings.Contains(resp.Error.Message, tc.wantMessage) {
				t.Fatalf("Error.Message = %q, want to contain %q", resp.Error.Message, tc.wantMessage)
			}
		})
	}
}

func TestHandleRequestReturnsSuccessForPartialAggregation(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	if err := (&config.Config{AccountKey: testAccountKey}).SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeJSONResponse(t, w, &api.HealthTypesResponse{
				Aggregated: []api.HealthTypeSection{
					{
						Name: "Activity",
						Types: []api.HealthType{
							{ID: 9, Name: "Step count", Category: "Cumulative", SubCategory: "Activity"},
						},
					},
				},
				Record: []api.HealthTypeSection{
					{
						Name: "Body",
						Types: []api.HealthType{
							{ID: 0, Name: "Body mass", Category: "Record", SubCategory: "Body"},
						},
					},
				},
			})
		case "/healthdata/encrypted":
			writeJSONResponse(t, w, []api.EncryptedPackage{
				{
					Type: 9,
					Data: []api.EncryptedUnitGroup{
						{
							Units: "count",
							Records: []api.EncryptedRecord{
								{Time: "2026-03-20T12:00:00Z", Nonce: "AQIDBAUGBwgJCgsM", Cipher: "lmaYhg=="},
								{Time: "2026-03-20T18:00:00Z", Nonce: "DAsKCQgHBgUEAwIB", Cipher: "wEQM8w=="},
							},
						},
					},
				},
				{
					Type: 0,
					Data: []api.EncryptedUnitGroup{
						{
							Units: "kg",
							Records: []api.EncryptedRecord{
								{Time: "2026-03-20T08:00:00Z", Nonce: "AQIDBAUGBwgJCgsM", Cipher: "lmaYhg=="},
							},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(apiServer.Close)

	server := NewServer(service.Options{
		ConfigPath: configPath,
		APIURL:     apiServer.URL,
	}, "1.0.0", "1.0.0", bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	resp, shouldRespond := server.handleRequest(request{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "tools/call",
		Params: mustJSONRaw(t, map[string]any{
			"name": "fetch_health_data",
			"arguments": map[string]any{
				"types":     []any{"step_count", "body_mass"},
				"from":      "2026-03-20",
				"to":        "2026-03-26",
				"aggregate": "day",
			},
		}),
	})
	if !shouldRespond {
		t.Fatal("shouldRespond = false, want true")
	}

	if resp.Error != nil {
		t.Fatalf("Error = %#v, want nil", resp.Error)
	}

	result := resp.Result.(toolResult)
	payload := result.StructuredContent.(map[string]any)

	if _, ok := payload["aggregated_results"]; !ok {
		t.Fatalf("payload = %#v, want aggregated_results", payload)
	}

	if _, ok := payload["results"]; !ok {
		t.Fatalf("payload = %#v, want results", payload)
	}
}

func TestHandleToolCallStatusUsesConfigOverrideWithoutLeakingKey(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	if err := (&config.Config{
		AccountKey: testAccountKey,
		APIURL:     "https://example.com/api/v2",
	}).SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	server := NewServer(service.Options{ConfigPath: configPath}, "1.0.0", "1.0.0", bytes.NewBuffer(nil), bytes.NewBuffer(nil))
	result := server.HandleToolCall("health_export_status", map[string]any{})
	if result.IsError {
		t.Fatalf("IsError = true, want false with payload %#v", result.StructuredContent)
	}

	payload := result.StructuredContent.(*service.MCPStatus)
	if !payload.Authenticated {
		t.Fatal("Authenticated = false, want true")
	}

	if payload.ConfigPath != configPath {
		t.Fatalf("ConfigPath = %q, want %q", payload.ConfigPath, configPath)
	}

	if payload.AuthSource != configPath {
		t.Fatalf("AuthSource = %q, want %q", payload.AuthSource, configPath)
	}

	text := result.Content[0].Text
	if bytes.Contains([]byte(text), []byte(testAccountKey)) {
		t.Fatal("tool result leaked raw account key")
	}
}

func TestServeInitializeAndListTools(t *testing.T) {
	input := bytes.NewBufferString("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2024-11-05\"}}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}\n")
	var output bytes.Buffer

	server := NewServer(service.Options{}, "1.0.0", "1.0.0", input, &output)
	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}

	decoder := json.NewDecoder(&output)
	var initializeResp map[string]any
	if err := decoder.Decode(&initializeResp); err != nil {
		t.Fatalf("Decode(initialize) error = %v", err)
	}

	if got := initializeResp["jsonrpc"]; got != "2.0" {
		t.Fatalf("jsonrpc = %v, want 2.0", got)
	}

	result := initializeResp["result"].(map[string]any)
	instructions, ok := result["instructions"].(string)
	if !ok {
		t.Fatalf("instructions type = %T, want string", result["instructions"])
	}

	for _, keyword := range []string{"Apple Health", "steps", "heart rate", "sleep", "workouts", "calories"} {
		if !strings.Contains(instructions, keyword) {
			t.Fatalf("instructions = %q, want keyword %q", instructions, keyword)
		}
	}

	var listResp map[string]any
	if err := decoder.Decode(&listResp); err != nil {
		t.Fatalf("Decode(list) error = %v", err)
	}

	tools := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 3 {
		t.Fatalf("len(tools) = %d, want 3", len(tools))
	}

	toolDescriptions := make(map[string]string, len(tools))
	for _, rawTool := range tools {
		tool := rawTool.(map[string]any)
		toolDescriptions[tool["name"].(string)] = tool["description"].(string)
	}

	assertContainsAll(t, toolDescriptions["fetch_health_data"], []string{"Apple Health", "steps", "heart rate", "sleep", "active energy", "workouts"})
	assertContainsAll(t, toolDescriptions["list_health_types"], []string{"Apple Health", "steps", "heart rate", "sleep", "workouts", "Call this tool first"})
	assertContainsAll(t, toolDescriptions["health_export_status"], []string{"Apple Health", "steps", "heart rate", "sleep", "workout", "fitness"})
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, response any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}

func assertContainsAll(t *testing.T, value string, needles []string) {
	t.Helper()

	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			t.Fatalf("value %q does not contain %q", value, needle)
		}
	}
}

func mustJSONRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	return body
}
