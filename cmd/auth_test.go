package cmd

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/config"
)

const testAccountKey = "abcdef.0123456789abcdef0123456789abcdef.gh01"

func TestAuthHelpCommands(t *testing.T) {
	resetViperState(t)
	setCmdConfigHome(t)

	cases := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "auth help",
			args:     []string{"auth", "--help"},
			contains: []string{"Manage the HealthExport account key used to decrypt data locally.", "Settings > Data sharing", "he auth login", "he auth status"},
		},
		{
			name: "auth login help",
			args: []string{"auth", "login", "--help"},
			contains: []string{
				"Settings > Data sharing",
				"HEALTHEXPORT_ACCOUNT_KEY",
				"~/.config/healthexport/config.yaml",
				"he data --type step_count --from 2024-01-01 --to 2024-01-31",
			},
		},
		{
			name:     "auth status help",
			args:     []string{"auth", "status", "--help"},
			contains: []string{"Show whether an account key is currently available to the CLI.", "masked key", "he auth status --account-key 0123456789abcdef0123456789abcdef"},
		},
		{
			name:     "auth logout help",
			args:     []string{"auth", "logout", "--help"},
			contains: []string{"Remove the saved account key from local config.", "HEALTHEXPORT_ACCOUNT_KEY", "he auth status"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restore := snapshotVersionState()
			t.Cleanup(restore)

			rootCmd.SetArgs(tc.args)

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			output := stdout.String() + stderr.String()
			for _, want := range tc.contains {
				assertContains(t, output, want)
			}
		})
	}
}

func TestLoginWithKeySavesConfigAndPrintsUID(t *testing.T) {
	setCmdConfigHome(t)

	var stderr bytes.Buffer
	if err := loginWithKey(testAccountKey, &stderr); err != nil {
		t.Fatalf("loginWithKey() error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AccountKey != testAccountKey {
		t.Fatalf("stored account key = %q, want %q", cfg.AccountKey, testAccountKey)
	}

	parsed, err := auth.Parse(testAccountKey)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assertContains(t, stderr.String(), "Account key saved to "+config.ConfigPath())
	assertContains(t, stderr.String(), "UID: "+parsed.UID)
}

func TestLoginWithKeyRejectsInvalidKey(t *testing.T) {
	setCmdConfigHome(t)

	err := loginWithKey("bad", ioDiscard{})
	if !errors.Is(err, auth.ErrInvalidKeyFormat) {
		t.Fatalf("loginWithKey() error = %v, want ErrInvalidKeyFormat", err)
	}
}

func TestStatusWithKeyPrintsResolvedAuthState(t *testing.T) {
	setCmdConfigHome(t)

	if err := (&config.Config{AccountKey: testAccountKey}).Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stderr bytes.Buffer
	if err := statusWithKey("", &stderr); err != nil {
		t.Fatalf("statusWithKey() error = %v", err)
	}

	parsed, err := auth.Parse(testAccountKey)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	assertContains(t, stderr.String(), "Authenticated")
	assertContains(t, stderr.String(), "  Account key: "+parsed.MaskedKey())
	assertContains(t, stderr.String(), "  UID: "+parsed.UID)
	assertContains(t, stderr.String(), "  Source: ~/.config/healthexport/config.yaml")
}

func TestStatusWithKeyReturnsExit2WhenNotAuthenticated(t *testing.T) {
	setCmdConfigHome(t)

	var stderr bytes.Buffer
	err := statusWithKey("", &stderr)
	if !errors.Is(err, auth.ErrNoAccountKey) {
		t.Fatalf("statusWithKey() error = %v, want ErrNoAccountKey", err)
	}

	if got := exitCodeForError(err); got != 2 {
		t.Fatalf("exitCodeForError() = %d, want 2", got)
	}

	if shouldPrintError(err) {
		t.Fatal("shouldPrintError() = true, want false")
	}

	assertContains(t, stderr.String(), "Not authenticated")
	assertContains(t, stderr.String(), "Run 'he auth login', set HEALTHEXPORT_ACCOUNT_KEY, or pass --account-key.")
}

func TestLogoutFromConfigRemovesOnlyAccountKey(t *testing.T) {
	setCmdConfigHome(t)

	cfg := &config.Config{
		AccountKey: testAccountKey,
		Format:     "json",
		APIURL:     "https://example.com/api/v2",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stderr bytes.Buffer
	if err := logoutFromConfig(&stderr); err != nil {
		t.Fatalf("logoutFromConfig() error = %v", err)
	}

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.AccountKey != "" {
		t.Fatalf("AccountKey = %q, want empty", got.AccountKey)
	}

	if got.Format != "json" {
		t.Fatalf("Format = %q, want %q", got.Format, "json")
	}

	if got.APIURL != "https://example.com/api/v2" {
		t.Fatalf("APIURL = %q, want %q", got.APIURL, "https://example.com/api/v2")
	}

	assertContains(t, stderr.String(), "Account key removed from "+config.ConfigPath())
}

func TestLogoutFromConfigWithoutStoredKeyPrintsNotice(t *testing.T) {
	setCmdConfigHome(t)

	var stderr bytes.Buffer
	if err := logoutFromConfig(&stderr); err != nil {
		t.Fatalf("logoutFromConfig() error = %v", err)
	}

	assertContains(t, stderr.String(), "No account key in config")

	if _, err := os.Stat(config.ConfigPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("config file exists unexpectedly, stat error = %v", err)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

func setCmdConfigHome(t *testing.T) {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", configHome)
}
