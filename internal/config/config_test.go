package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadNonExistentReturnsZeroConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := *cfg, (Config{}); got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestSaveCreatesDirectoryAndFileWithExpectedPermissions(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)

	cfg := &Config{AccountKey: "abc123"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	dirInfo, err := os.Stat(ConfigDir())
	if err != nil {
		t.Fatalf("Stat(config dir) error = %v", err)
	}

	if got, want := dirInfo.Mode().Perm(), os.FileMode(0o700); got != want {
		t.Fatalf("config dir perms = %#o, want %#o", got, want)
	}

	fileInfo, err := os.Stat(ConfigPath())
	if err != nil {
		t.Fatalf("Stat(config file) error = %v", err)
	}

	if got, want := fileInfo.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("config file perms = %#o, want %#o", got, want)
	}
}

func TestSaveLoadRoundTripPreservesAllFields(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)

	want := &Config{
		AccountKey: "abc123.1234567890abcdef1234567890abcdef.def4",
		Format:     "json",
		APIURL:     "https://example.com/api/v2",
	}

	if err := want.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestSaveLoadRoundTripPreservesPartialConfig(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)

	want := &Config{
		Format: "csv",
	}

	if err := want.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}

func TestSetFieldWithValidKeysWorks(t *testing.T) {
	cfg := &Config{}

	cases := []struct {
		key   string
		value string
		want  string
	}{
		{key: "account_key", value: "abc123", want: "abc123"},
		{key: "format", value: "json", want: "json"},
		{key: "api_url", value: "https://example.com/api/v2", want: "https://example.com/api/v2"},
	}

	for _, tc := range cases {
		if err := cfg.SetField(tc.key, tc.value); err != nil {
			t.Fatalf("SetField(%q, %q) error = %v", tc.key, tc.value, err)
		}

		got, err := cfg.GetField(tc.key)
		if err != nil {
			t.Fatalf("GetField(%q) error = %v", tc.key, err)
		}

		if got != tc.want {
			t.Fatalf("GetField(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestSetFieldUnknownKeyReturnsError(t *testing.T) {
	cfg := &Config{}

	if err := cfg.SetField("unknown", "value"); err == nil {
		t.Fatal("SetField() error = nil, want error")
	}
}

func TestSetFieldInvalidFormatReturnsError(t *testing.T) {
	cfg := &Config{}

	if err := cfg.SetField("format", "xml"); err == nil {
		t.Fatal("SetField() error = nil, want error")
	}
}

func TestSetFieldInvalidAPIURLReturnsError(t *testing.T) {
	cfg := &Config{}

	if err := cfg.SetField("api_url", "ftp://example.com"); err == nil {
		t.Fatal("SetField() error = nil, want error")
	}
}

func TestGetFieldReturnsCorrectValues(t *testing.T) {
	cfg := &Config{
		AccountKey: "abc123",
		Format:     "csv",
		APIURL:     "https://example.com/api/v2",
	}

	cases := []struct {
		key  string
		want string
	}{
		{key: "account_key", want: "abc123"},
		{key: "format", want: "csv"},
		{key: "api_url", want: "https://example.com/api/v2"},
	}

	for _, tc := range cases {
		got, err := cfg.GetField(tc.key)
		if err != nil {
			t.Fatalf("GetField(%q) error = %v", tc.key, err)
		}

		if got != tc.want {
			t.Fatalf("GetField(%q) = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestGetFieldUnknownKeyReturnsError(t *testing.T) {
	cfg := &Config{}

	if _, err := cfg.GetField("unknown"); err == nil {
		t.Fatal("GetField() error = nil, want error")
	}
}

func TestValidKeysReturnsExpectedList(t *testing.T) {
	got := ValidKeys()
	want := []string{"account_key", "format", "api_url"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ValidKeys() = %#v, want %#v", got, want)
	}
}

func TestConfigPathUsesConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)

	want := filepath.Join(configHome, "healthexport", "config.yaml")
	if got := ConfigPath(); got != want {
		t.Fatalf("ConfigPath() = %q, want %q", got, want)
	}
}
