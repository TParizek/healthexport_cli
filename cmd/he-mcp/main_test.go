package main

import (
	"log"
	"testing"
)

func TestSanitizeOptionalEnv(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "whitespace", input: "   ", want: ""},
		{name: "placeholder", input: "${user_config.apiURL}", want: ""},
		{name: "null", input: "null", want: ""},
		{name: "undefined", input: "undefined", want: ""},
		{name: "value", input: "https://example.com/api/v2", want: "https://example.com/api/v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeOptionalEnv(tt.input); got != tt.want {
				t.Fatalf("sanitizeOptionalEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseIntEnvIgnoresPlaceholderValue(t *testing.T) {
	t.Setenv("HE_MCP_REQUEST_TIMEOUT_SECONDS", "${user_config.requestTimeoutSeconds}")

	got := parseIntEnv("HE_MCP_REQUEST_TIMEOUT_SECONDS", 30, log.New(testWriter{t}, "", 0))
	if got != 30 {
		t.Fatalf("parseIntEnv() = %d, want 30", got)
	}
}

type testWriter struct {
	t *testing.T
}

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}
