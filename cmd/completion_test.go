package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionHelpIncludesShellInstructions(t *testing.T) {
	resetViperState(t)
	setCmdConfigHome(t)

	stdout, stderr, err := executeRootCommand(t, []string{"completion", "--help"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout + stderr
	assertContains(t, output, "Generate a shell completion script for he.")
	assertContains(t, output, "source <(he completion bash)")
	assertContains(t, output, `he completion zsh > "${fpath[1]}/_he"`)
}

func TestRunCompletionBashWritesScript(t *testing.T) {
	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	if err := runCompletion(cmd, []string{"bash"}); err != nil {
		t.Fatalf("runCompletion() error = %v", err)
	}

	output := stdout.String()
	assertContains(t, output, "bash completion V2 for he")
	assertContains(t, output, "__start_he")
}

func TestFlagCompletionsProvideKnownValues(t *testing.T) {
	tests := []struct {
		cmd      *cobra.Command
		flagName string
		want     []string
	}{
		{cmd: dataCmd, flagName: "format", want: []string{"csv", "json"}},
		{cmd: dataCmd, flagName: "aggregate", want: []string{"day", "week", "month", "year"}},
		{cmd: typesCmd, flagName: "format", want: []string{"csv", "json"}},
		{cmd: typesCmd, flagName: "category", want: []string{"aggregated", "record", "workout"}},
	}

	for _, tc := range tests {
		completionFunc, ok := tc.cmd.GetFlagCompletionFunc(tc.flagName)
		if !ok {
			t.Fatalf("GetFlagCompletionFunc(%q) = false, want true", tc.flagName)
		}

		got, directive := completionFunc(tc.cmd, nil, "")
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Fatalf("completion directive = %v, want %v", directive, cobra.ShellCompDirectiveNoFileComp)
		}

		if len(got) != len(tc.want) {
			t.Fatalf("completion values = %v, want %v", got, tc.want)
		}

		for i, want := range tc.want {
			if got[i] != want {
				t.Fatalf("completion values = %v, want %v", got, tc.want)
			}
		}
	}
}

func TestTypeFlagHasNoStaticCompletion(t *testing.T) {
	if _, ok := dataCmd.GetFlagCompletionFunc("type"); ok {
		t.Fatal("GetFlagCompletionFunc(type) = true, want false")
	}
}
