package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/config"
)

func TestMCPHelp(t *testing.T) {
	resetViperState(t)
	setCmdConfigHome(t)
	restore := snapshotVersionState()
	t.Cleanup(restore)

	rootCmd.SetArgs([]string{"mcp", "--help"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout.String() + stderr.String()
	assertContains(t, output, "Inspect HealthExport MCP integration status and local diagnostics.")
	assertContains(t, output, "he mcp status --format json")
}

func TestRunMCPStatusJSON(t *testing.T) {
	setCmdConfigHome(t)

	if err := (&config.Config{
		AccountKey: testAccountKey,
		APIURL:     "https://example.com/api/v2",
	}).Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runMCPStatus("json", &stdout, &stderr); err != nil {
		t.Fatalf("runMCPStatus() error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got, want := payload["he_version"], version; got != want {
		t.Fatalf("he_version = %v, want %v", got, want)
	}

	if got, want := payload["authenticated"], true; got != want {
		t.Fatalf("authenticated = %v, want %v", got, want)
	}

	if got, want := payload["auth_source"], "~/.config/healthexport/config.yaml"; got != want {
		t.Fatalf("auth_source = %v, want %v", got, want)
	}

	if got, want := payload["config_path"], "~/.config/healthexport/config.yaml"; got != want {
		t.Fatalf("config_path = %v, want %v", got, want)
	}

	if got, want := payload["api_url"], "https://example.com/api/v2"; got != want {
		t.Fatalf("api_url = %v, want %v", got, want)
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
