package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts for he",
	Long: strings.TrimSpace(`
Generate a shell completion script for he.

To load completions:

Bash:
  $ source <(he completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ he completion bash > /etc/bash_completion.d/he
  # macOS:
  $ he completion bash > $(brew --prefix)/etc/bash_completion.d/he

Zsh:
  $ source <(he completion zsh)
  # To load completions for each session:
  $ he completion zsh > "${fpath[1]}/_he"

Fish:
  $ he completion fish | source
  # To load completions for each session:
  $ he completion fish > ~/.config/fish/completions/he.fish

PowerShell:
  PS> he completion powershell | Out-String | Invoke-Expression
`),
	Example: strings.TrimSpace(`
  # Load Bash completions for the current shell
  source <(he completion bash)

  # Write Zsh completions to the first directory in fpath
  he completion zsh > "${fpath[1]}/_he"
`),
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []cobra.Completion{"bash", "zsh", "fish", "powershell"},
	RunE:      runCompletion,
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(completionCmd)
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "bash":
		return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
	case "zsh":
		return rootCmd.GenZshCompletion(cmd.OutOrStdout())
	case "fish":
		return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
	case "powershell":
		return rootCmd.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
	default:
		return fmt.Errorf("unsupported shell %q (valid shells: bash, zsh, fish, powershell)", args[0])
	}
}
