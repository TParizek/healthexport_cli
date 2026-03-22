package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestConfigSetHelpIncludesValidKeysAndExamples(t *testing.T) {
	resetViperState(t)
	setCmdConfigHome(t)

	stdout, stderr, err := executeRootCommand(t, []string{"config", "set", "--help"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout + stderr
	assertContains(t, output, "Set a configuration value.")
	assertContains(t, output, "format       Default output format (csv, json)")
	assertContains(t, output, "api_url      API base URL")
	assertContains(t, output, "account_key  Account key (prefer 'he auth login' instead)")
	assertContains(t, output, "Set JSON as the default output format")
	assertContains(t, output, "Point the CLI at a custom API base URL")
	assertContains(t, output, "he config set format json")
	assertContains(t, output, "he config set api_url https://custom.example.com/api/v2")
}

func TestConfigHelpIncludesExamples(t *testing.T) {
	resetViperState(t)
	setCmdConfigHome(t)

	stdout, stderr, err := executeRootCommand(t, []string{"config", "--help"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout + stderr
	assertContains(t, output, "Inspect and update local CLI configuration stored on this machine.")
	assertContains(t, output, "he config list")
	assertContains(t, output, "he config get api_url")
}

func TestConfigGetHelpIncludesDefaults(t *testing.T) {
	resetViperState(t)
	setCmdConfigHome(t)

	stdout, stderr, err := executeRootCommand(t, []string{"config", "get", "--help"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout + stderr
	assertContains(t, output, "Get a single configuration value by key.")
	assertContains(t, output, "built-in default")
	assertContains(t, output, "he config get format")
}

func TestConfigListHelpIncludesMaskingNotice(t *testing.T) {
	resetViperState(t)
	setCmdConfigHome(t)

	stdout, stderr, err := executeRootCommand(t, []string{"config", "list", "--help"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout + stderr
	assertContains(t, output, "Stored account keys are masked in the output")
	assertContains(t, output, "he config list")
}

func TestConfigSetValidKeyUpdatesConfigFile(t *testing.T) {
	setCmdConfigHome(t)

	var stderr bytes.Buffer
	if err := runConfigSet("format", "json", &stderr); err != nil {
		t.Fatalf("runConfigSet() error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got, want := cfg.Format, "json"; got != want {
		t.Fatalf("Format = %q, want %q", got, want)
	}

	assertContains(t, stderr.String(), "Config updated: format = json")
}

func TestConfigSetUnknownKeyReturnsExit4(t *testing.T) {
	setCmdConfigHome(t)

	err := runConfigSet("unknown", "value", ioDiscard{})
	if err == nil {
		t.Fatal("runConfigSet() error = nil, want error")
	}

	if got := exitCodeForError(err); got != 4 {
		t.Fatalf("exitCodeForError() = %d, want 4", got)
	}

	if !shouldPrintError(err) {
		t.Fatal("shouldPrintError() = false, want true")
	}

	assertContains(t, err.Error(), `unknown config key "unknown"`)
}

func TestConfigSetInvalidValueReturnsExit4(t *testing.T) {
	setCmdConfigHome(t)

	err := runConfigSet("format", "xml", ioDiscard{})
	if err == nil {
		t.Fatal("runConfigSet() error = nil, want error")
	}

	if got := exitCodeForError(err); got != 4 {
		t.Fatalf("exitCodeForError() = %d, want 4", got)
	}

	assertContains(t, err.Error(), `invalid format "xml": must be csv or json`)
}

func TestConfigSetWrongArgCountReturnsError(t *testing.T) {
	resetViperState(t)
	setCmdConfigHome(t)

	_, _, err := executeRootCommand(t, []string{"config", "set", "format"})
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}

	assertContains(t, err.Error(), "accepts 2 arg(s), received 1")
}

func TestConfigGetReturnsStoredValueAndDefaults(t *testing.T) {
	setCmdConfigHome(t)

	cfg := &config.Config{Format: "json"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout bytes.Buffer
	if err := runConfigGet("format", &stdout); err != nil {
		t.Fatalf("runConfigGet(format) error = %v", err)
	}

	if got, want := stdout.String(), "json\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	stdout.Reset()
	if err := runConfigGet("api_url", &stdout); err != nil {
		t.Fatalf("runConfigGet(api_url) error = %v", err)
	}

	if got, want := stdout.String(), config.DefaultAPIURL+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestConfigGetUnknownKeyReturnsExit4(t *testing.T) {
	setCmdConfigHome(t)

	err := runConfigGet("unknown", ioDiscard{})
	if err == nil {
		t.Fatal("runConfigGet() error = nil, want error")
	}

	if got := exitCodeForError(err); got != 4 {
		t.Fatalf("exitCodeForError() = %d, want 4", got)
	}
}

func TestConfigListShowsAllFieldsAndMasksAccountKey(t *testing.T) {
	setCmdConfigHome(t)

	cfg := &config.Config{
		AccountKey: testAccountKey,
		Format:     "json",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout bytes.Buffer
	if err := runConfigList(&stdout); err != nil {
		t.Fatalf("runConfigList() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if got, want := len(lines), 3; got != want {
		t.Fatalf("line count = %d, want %d", got, want)
	}

	assertContains(t, stdout.String(), "account_key=abcdef.******************************.gh01\n")
	assertContains(t, stdout.String(), "format=json\n")
	assertContains(t, stdout.String(), "api_url="+config.DefaultAPIURL+"\n")
}

func TestVersionCommandPrintsBuildMetadata(t *testing.T) {
	restore := snapshotVersionState()
	t.Cleanup(restore)

	SetVersionInfo("1.2.0", "abc1234", "2024-06-15T10:30:00Z")

	stdout, _, err := executeRootCommand(t, []string{"version"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got, want := stdout, "he version 1.2.0 (commit abc1234, built 2024-06-15T10:30:00Z)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func executeRootCommand(t *testing.T, args []string) (string, string, error) {
	t.Helper()

	restore := snapshotVersionState()
	t.Cleanup(restore)
	resetCommandFlags(rootCmd)

	rootCmd.SetArgs(args)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

func resetCommandFlags(cmd *cobra.Command) {
	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())

	for _, child := range cmd.Commands() {
		resetCommandFlags(child)
	}
}

func resetFlagSet(flagSet *pflag.FlagSet) {
	flagSet.VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})
}
