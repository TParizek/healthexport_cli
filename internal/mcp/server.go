package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/service"
)

const DefaultProtocolVersion = "2024-11-05"

const serverInstructions = "HealthExport provides access to Apple Health data including steps, heart rate, sleep, workouts, calories, weight, nutrition, blood pressure, Apple Watch activity, and many other health and fitness metrics. Use these tools whenever the user asks about health data, fitness stats, activity history, biometric measurements, sleep, workouts, or Apple Health trends."

type Server struct {
	options              service.Options
	version              string
	compatibleCLIVersion string
	in                   io.Reader
	out                  io.Writer
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type toolDefinition struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
	Annotations  map[string]any `json:"annotations,omitempty"`
}

type toolResult struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type listHealthTypesArgs struct {
	Category string `json:"category,omitempty"`
}

type fetchHealthDataArgs struct {
	Types     []string `json:"types"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Aggregate string   `json:"aggregate,omitempty"`
}

type mcpHealthType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`
}

func NewServer(opts service.Options, version, compatibleCLIVersion string, in io.Reader, out io.Writer) *Server {
	return &Server{
		options:              opts,
		version:              version,
		compatibleCLIVersion: compatibleCLIVersion,
		in:                   in,
		out:                  out,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	decoder := json.NewDecoder(s.in)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req request
		if err := decoder.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		resp, shouldRespond := s.handleRequest(req)
		if !shouldRespond {
			continue
		}

		if err := s.writeResponse(resp); err != nil {
			return err
		}
	}
}

func (s *Server) HandleToolCall(name string, arguments map[string]any) toolResult {
	switch name {
	case "list_health_types":
		return s.handleListHealthTypes(arguments)
	case "fetch_health_data":
		return s.handleFetchHealthData(arguments)
	case "health_export_status":
		return s.handleHealthExportStatus(arguments)
	default:
		return s.toolError("invalid_input", fmt.Sprintf("unknown tool %q", name))
	}
}

func ToolDefinitions() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "list_health_types",
			Description: "HealthExport healthexport he — List all available health and fitness metric types from Apple Health. Returns categories of queryable metrics including: steps, step count, walking, running distance, cycling distance, flights climbed, active calories, resting calories, exercise minutes, heart rate, resting heart rate, HRV, blood pressure, blood oxygen, respiratory rate, body temperature, weight, BMI, body fat, sleep, sleep analysis, nutrition, water intake, caffeine, workouts, and more. Call this tool first before fetching any health data to discover which metrics are available. Supports filtering by category: 'aggregated' (totals like steps and calories), 'record' (readings like heart rate and weight), or 'workout' (exercise sessions).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type":        "string",
						"description": "Optional category filter: 'aggregated' for totals like steps, calories, and distance; 'record' for readings like heart rate, blood pressure, sleep, or weight; or 'workout' for exercise sessions like running, cycling, and swimming.",
						"enum":        []string{"aggregated", "record", "workout"},
					},
				},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{"type": "string"},
					"types": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id":          map[string]any{"type": "integer"},
								"name":        map[string]any{"type": "string"},
								"category":    map[string]any{"type": "string"},
								"subcategory": map[string]any{"type": "string"},
							},
							"required":             []string{"id", "name", "category", "subcategory"},
							"additionalProperties": false,
						},
					},
				},
				"required":             []string{"types"},
				"additionalProperties": false,
			},
			Annotations: map[string]any{
				"readOnlyHint":  true,
				"openWorldHint": true,
			},
		},
		{
			Name:        "fetch_health_data",
			Description: "HealthExport healthexport he — Fetch health and fitness data from Apple Health for a specific date range. Query any metric including steps, step count, walking distance, running distance, cycling distance, flights climbed, active energy, resting energy, exercise time, heart rate, resting heart rate, HRV, blood pressure, blood oxygen, weight, BMI, body fat, sleep analysis, water intake, nutrition, caffeine, workouts, and more. Results can be aggregated by day, week, month, or year for trends and summaries. Use list_health_types first to discover all available metric names.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"types": map[string]any{
						"type":        "array",
						"description": "Health metric names such as 'Step count', 'Heart rate', 'Active energy burned', 'Sleep analysis', or numeric type IDs. Call list_health_types to see all available options.",
						"minItems":    1,
						"items": map[string]any{
							"anyOf": []map[string]any{
								{"type": "string"},
								{"type": "integer"},
							},
						},
					},
					"from": map[string]any{
						"type":        "string",
						"description": "Start date in YYYY-MM-DD or RFC3339 format for the health data query.",
					},
					"to": map[string]any{
						"type":        "string",
						"description": "End date in YYYY-MM-DD or RFC3339 format for the health data query.",
					},
					"aggregate": map[string]any{
						"type":        "string",
						"description": "Optional summary period for trends and totals: day, week, month, or year.",
						"enum":        []string{"day", "week", "month", "year"},
					},
				},
				"required":             []string{"types", "from", "to"},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from":               map[string]any{"type": "string"},
					"to":                 map[string]any{"type": "string"},
					"aggregate":          map[string]any{"type": "string"},
					"resolved_type_ids":  map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
					"warnings":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"results":            map[string]any{"type": "array"},
					"aggregated_results": map[string]any{"type": "array"},
				},
				"required":             []string{"from", "to"},
				"additionalProperties": false,
			},
			Annotations: map[string]any{
				"readOnlyHint":  true,
				"openWorldHint": true,
			},
		},
		{
			Name:        "health_export_status",
			Description: "HealthExport healthexport he — Check whether the HealthExport MCP server is connected and ready to query Apple Health data. Call this to verify connectivity before fetching steps, heart rate, sleep, weight, workouts, calories, or any other health and fitness metrics.",
			InputSchema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
			OutputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"server_version":         map[string]any{"type": "string"},
					"compatible_cli_version": map[string]any{"type": "string"},
					"he_version":             map[string]any{"type": "string"},
					"authenticated":          map[string]any{"type": "boolean"},
					"auth_source":            map[string]any{"type": "string"},
					"config_path":            map[string]any{"type": "string"},
					"api_url":                map[string]any{"type": "string"},
				},
				"required":             []string{"server_version", "compatible_cli_version", "he_version", "authenticated", "auth_source", "config_path", "api_url"},
				"additionalProperties": false,
			},
			Annotations: map[string]any{
				"readOnlyHint": true,
			},
		},
	}
}

func (s *Server) handleRequest(req request) (response, bool) {
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return s.errorResponse(req.ID, -32600, "invalid request", nil), len(req.ID) > 0
	}

	switch req.Method {
	case "initialize":
		var params initializeParams
		if err := decodeParams(req.Params, &params); err != nil {
			return s.errorResponse(req.ID, -32602, err.Error(), nil), true
		}

		protocolVersion := strings.TrimSpace(params.ProtocolVersion)
		if protocolVersion == "" {
			protocolVersion = DefaultProtocolVersion
		}

		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"instructions": serverInstructions,
				"serverInfo": map[string]any{
					"name":    "health-export",
					"version": s.version,
				},
			},
		}, true
	case "notifications/initialized":
		return response{}, false
	case "ping":
		return response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}, true
	case "tools/list":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"tools": ToolDefinitions(),
			},
		}, true
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := decodeParams(req.Params, &params); err != nil {
			return s.errorResponse(req.ID, -32602, err.Error(), nil), true
		}

		result := s.HandleToolCall(params.Name, params.Arguments)
		if result.IsError {
			errResp := toolResultToResponseError(result)
			return s.errorResponse(req.ID, errResp.Code, errResp.Message, errResp.Data), true
		}

		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}, true
	default:
		if len(req.ID) == 0 {
			return response{}, false
		}

		return s.errorResponse(req.ID, -32601, fmt.Sprintf("method %q not found", req.Method), nil), true
	}
}

func (s *Server) handleListHealthTypes(arguments map[string]any) toolResult {
	args, err := decodeArguments[listHealthTypesArgs](arguments, map[string]struct{}{
		"category": {},
	})
	if err != nil {
		return s.toolError("invalid_input", err.Error())
	}

	types, err := service.ListHealthTypes(s.options, args.Category)
	if err != nil {
		return s.mapToolError(err)
	}

	shapedTypes := make([]mcpHealthType, 0, len(types))
	for _, healthType := range types {
		shapedTypes = append(shapedTypes, mcpHealthType{
			ID:          healthType.ID,
			Name:        healthType.Name,
			Category:    healthType.Category,
			Subcategory: healthType.SubCategory,
		})
	}

	payload := map[string]any{
		"types": shapedTypes,
	}
	if strings.TrimSpace(args.Category) != "" {
		payload["category"] = strings.ToLower(strings.TrimSpace(args.Category))
	}

	return s.toolSuccess(payload)
}

func (s *Server) handleFetchHealthData(arguments map[string]any) toolResult {
	args, err := decodeFetchArguments(arguments)
	if err != nil {
		return s.toolError("invalid_input", err.Error())
	}

	result, err := service.FetchHealthData(s.options, service.FetchRequest{
		Types:                   args.Types,
		From:                    args.From,
		To:                      args.To,
		Aggregate:               args.Aggregate,
		AllowPartialAggregation: true,
	})
	if err != nil {
		return s.mapToolError(err)
	}

	payload := map[string]any{
		"from":              result.From,
		"to":                result.To,
		"resolved_type_ids": result.ResolvedTypeIDs,
	}
	if len(result.Warnings) > 0 {
		payload["warnings"] = result.Warnings
	}
	if result.Aggregate != "" {
		payload["aggregate"] = result.Aggregate
		payload["aggregated_results"] = result.Aggregated
	}
	if len(result.Decrypted) > 0 {
		payload["results"] = result.Decrypted
	}

	return s.toolSuccess(payload)
}

func (s *Server) handleHealthExportStatus(arguments map[string]any) toolResult {
	if len(arguments) > 0 {
		return s.toolError("invalid_input", "health_export_status does not accept arguments")
	}

	status, err := service.GetMCPStatus(s.options, s.version, s.version, s.compatibleCLIVersion)
	if err != nil {
		return s.mapToolError(err)
	}

	return s.toolSuccess(status)
}

func (s *Server) toolSuccess(payload any) toolResult {
	body, err := json.Marshal(payload)
	if err != nil {
		return s.toolError("internal_error", err.Error())
	}

	return toolResult{
		StructuredContent: payload,
		Content: []toolContent{
			{
				Type: "text",
				Text: string(body),
			},
		},
	}
}

func (s *Server) mapToolError(err error) toolResult {
	switch {
	case errors.Is(err, auth.ErrNoAccountKey):
		return s.toolError("auth_missing", "authentication is not configured on the host machine; run 'he auth login'")
	case errors.Is(err, service.ErrInvalidInput):
		return s.toolError("invalid_input", err.Error())
	case errors.Is(err, service.ErrConfig):
		return s.toolError("configuration_error", err.Error())
	default:
		var apiErr *api.APIError
		if errors.As(err, &apiErr) {
			return s.toolError("api_error", apiErr.Error())
		}

		return s.toolError("internal_error", err.Error())
	}
}

func (s *Server) toolError(category, message string) toolResult {
	payload := map[string]any{
		"error": map[string]any{
			"category": category,
			"message":  message,
		},
	}

	return toolResult{
		IsError:           true,
		StructuredContent: payload,
		Content: []toolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("%s: %s", category, message),
			},
		},
	}
}

func toolResultToResponseError(result toolResult) responseError {
	payload, ok := result.StructuredContent.(map[string]any)
	if !ok {
		return responseError{
			Code:    -32603,
			Message: "internal server error",
		}
	}

	errorPayload, ok := payload["error"].(map[string]any)
	if !ok {
		return responseError{
			Code:    -32603,
			Message: "internal server error",
		}
	}

	category, _ := errorPayload["category"].(string)
	message, _ := errorPayload["message"].(string)
	if strings.TrimSpace(message) == "" {
		message = "internal server error"
	}

	code := -32603
	switch category {
	case "invalid_input":
		code = -32602
	case "auth_missing":
		code = -32001
	case "configuration_error":
		code = -32002
	case "api_error":
		code = -32003
	}

	return responseError{
		Code:    code,
		Message: message,
		Data: map[string]any{
			"category": category,
		},
	}
}

func (s *Server) writeResponse(resp response) error {
	encoder := json.NewEncoder(s.out)
	return encoder.Encode(resp)
}

func (s *Server) errorResponse(id json.RawMessage, code int, message string, data any) response {
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &responseError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func decodeParams(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return nil
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	return nil
}

func decodeArguments[T any](arguments map[string]any, allowed map[string]struct{}) (T, error) {
	var target T
	for key := range arguments {
		if _, ok := allowed[key]; !ok {
			return target, fmt.Errorf("unexpected argument %q", key)
		}
	}

	body, err := json.Marshal(arguments)
	if err != nil {
		return target, err
	}

	if err := json.Unmarshal(body, &target); err != nil {
		return target, fmt.Errorf("invalid arguments: %w", err)
	}

	return target, nil
}

func decodeFetchArguments(arguments map[string]any) (fetchHealthDataArgs, error) {
	for key := range arguments {
		switch key {
		case "types", "from", "to", "aggregate":
		default:
			return fetchHealthDataArgs{}, fmt.Errorf("unexpected argument %q", key)
		}
	}

	var args fetchHealthDataArgs
	args.From = stringValue(arguments["from"])
	args.To = stringValue(arguments["to"])
	args.Aggregate = stringValue(arguments["aggregate"])

	rawTypes, ok := arguments["types"]
	if !ok {
		return args, fmt.Errorf("types is required")
	}

	items, ok := rawTypes.([]any)
	if !ok {
		return args, fmt.Errorf("types must be an array")
	}

	args.Types = make([]string, 0, len(items))
	for _, item := range items {
		switch value := item.(type) {
		case string:
			args.Types = append(args.Types, value)
		case float64:
			if value != float64(int(value)) {
				return args, fmt.Errorf("types must contain only strings or integers")
			}
			args.Types = append(args.Types, strconv.Itoa(int(value)))
		default:
			return args, fmt.Errorf("types must contain only strings or integers")
		}
	}

	return args, nil
}

func stringValue(value any) string {
	str, _ := value.(string)
	return str
}
