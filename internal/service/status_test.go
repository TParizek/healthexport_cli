package service

import (
	"testing"

	"github.com/TParizek/healthexport_cli/internal/config"
)

const serviceTestAccountKey = "abcdef.0123456789abcdef0123456789abcdef.gh01"

func TestGetStatusUsesConfigPathOverride(t *testing.T) {
	overridePath := t.TempDir() + "/custom-config.yaml"
	if err := (&config.Config{
		AccountKey: serviceTestAccountKey,
		APIURL:     "https://override.example.com/api/v2",
	}).SaveToPath(overridePath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	status, err := GetStatus(Options{ConfigPath: overridePath}, "1.2.3")
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if got, want := status.HEVersion, "1.2.3"; got != want {
		t.Fatalf("HEVersion = %q, want %q", got, want)
	}

	if !status.Authenticated {
		t.Fatal("Authenticated = false, want true")
	}

	if got, want := status.AuthSource, overridePath; got != want {
		t.Fatalf("AuthSource = %q, want %q", got, want)
	}

	if got, want := status.ConfigPath, overridePath; got != want {
		t.Fatalf("ConfigPath = %q, want %q", got, want)
	}

	if got, want := status.APIURL, "https://override.example.com/api/v2"; got != want {
		t.Fatalf("APIURL = %q, want %q", got, want)
	}
}
