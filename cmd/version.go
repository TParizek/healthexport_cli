package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print CLI version and build metadata",
	Long: strings.TrimSpace(`
Print the CLI version, commit, and build date.

This is useful for support, debugging, and confirming which build is installed
on the current machine.
`),
	Example: strings.TrimSpace(`
  # Print the full build metadata
  he version

  # Print the version using the global flag
  he --version
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "he version %s\n", versionString())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
