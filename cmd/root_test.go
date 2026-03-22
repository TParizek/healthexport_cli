package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestRootCommandHelp(t *testing.T) {
	restore := snapshotVersionState()
	t.Cleanup(restore)
	resetCommandFlags(rootCmd)

	rootCmd.SetArgs([]string{"--help"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	output := stdout.String() + stderr.String()

	assertContains(t, output, "HealthExport CLI - read and decrypt your health data.")
	assertContains(t, output, "Environment Variables:")
	assertContains(t, output, "HEALTHEXPORT_ACCOUNT_KEY")
	assertContains(t, output, "completion  Generate shell completion script")
}

func TestRootCommandPersistentFlags(t *testing.T) {
	if rootCmd.PersistentFlags().Lookup("account-key") == nil {
		t.Fatal("expected account-key persistent flag to be registered")
	}

	if rootCmd.PersistentFlags().Lookup("api-url") == nil {
		t.Fatal("expected api-url persistent flag to be registered")
	}
}

func TestSetVersionInfoUpdatesCommandVersion(t *testing.T) {
	restore := snapshotVersionState()
	t.Cleanup(restore)

	SetVersionInfo("1.2.3", "abc1234", "2026-03-23T10:00:00Z")

	if got, want := rootCmd.Version, "1.2.3 (commit abc1234, built 2026-03-23T10:00:00Z)"; got != want {
		t.Fatalf("rootCmd.Version = %q, want %q", got, want)
	}
}

func TestRootCommandVersionFlagPrintsBuildMetadata(t *testing.T) {
	restore := snapshotVersionState()
	t.Cleanup(restore)
	resetCommandFlags(rootCmd)

	SetVersionInfo("1.2.3", "abc1234", "2026-03-23T10:00:00Z")
	rootCmd.SetArgs([]string{"--version"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() returned error: %v", err)
	}

	if got, want := stdout.String(), "he version 1.2.3 (commit abc1234, built 2026-03-23T10:00:00Z)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestInitConfigLoadsValuesFromConfigFile(t *testing.T) {
	resetViperState(t)

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)

	configDir := filepath.Join(configHome, "healthexport")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	configBody := "account_key: stored-key\napi_url: https://example.com/api/v2\n"
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := initConfig(); err != nil {
		t.Fatalf("initConfig() error = %v", err)
	}

	if got, want := viper.GetString("account_key"), "stored-key"; got != want {
		t.Fatalf("viper.GetString(account_key) = %q, want %q", got, want)
	}

	if got, want := viper.GetString("api_url"), "https://example.com/api/v2"; got != want {
		t.Fatalf("viper.GetString(api_url) = %q, want %q", got, want)
	}
}

func TestInitConfigIgnoresMissingConfigFile(t *testing.T) {
	resetViperState(t)

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)

	if err := initConfig(); err != nil {
		t.Fatalf("initConfig() error = %v, want nil for missing config", err)
	}
}

func snapshotVersionState() func() {
	prevVersion := version
	prevCommit := commit
	prevDate := date
	prevRootVersion := rootCmd.Version
	prevAccountKey := accountKey
	prevAPIURL := apiURL

	return func() {
		version = prevVersion
		commit = prevCommit
		date = prevDate
		accountKey = prevAccountKey
		apiURL = prevAPIURL
		rootCmd.Version = prevRootVersion
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()

	if !strings.Contains(haystack, needle) {
		t.Fatalf("output %q does not contain %q", haystack, needle)
	}
}

func resetViperState(t *testing.T) {
	t.Helper()

	viper.Reset()
	viper.SetEnvPrefix("HEALTHEXPORT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	if err := viper.BindPFlag("account_key", rootCmd.PersistentFlags().Lookup("account-key")); err != nil {
		t.Fatalf("BindPFlag(account_key) error = %v", err)
	}

	if err := viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url")); err != nil {
		t.Fatalf("BindPFlag(api_url) error = %v", err)
	}

	t.Cleanup(func() {
		viper.Reset()
		viper.SetEnvPrefix("HEALTHEXPORT")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

		_ = viper.BindPFlag("account_key", rootCmd.PersistentFlags().Lookup("account-key"))
		_ = viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
	})
}
