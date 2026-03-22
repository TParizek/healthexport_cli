package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage local account key authentication",
	Long: strings.TrimSpace(`
Manage the HealthExport account key used to decrypt data locally.

Find your account key in the HealthExport app under Settings > Data sharing.
Use the subcommands below to save it to config, inspect the resolved key, or
remove the stored key.
`),
	Example: strings.TrimSpace(`
  # Save your account key to local config
  he auth login

  # Show which account key source is active
  he auth status

  # Remove the stored account key from config
  he auth logout
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save an account key locally",
	Long: strings.TrimSpace(`
Prompt for your HealthExport account key and save it to local config.

Find the account key in the HealthExport app under Settings > Data sharing.

Expected format:
  - abcdef.0123456789abcdef0123456789abcdef.gh01
  - 0123456789abcdef0123456789abcdef

The key is stored in ~/.config/healthexport/config.yaml.
You can also provide a key via HEALTHEXPORT_ACCOUNT_KEY or the --account-key flag.
`),
	Example: strings.TrimSpace(`
  # Save the key by entering it at the prompt
  he auth login

  # Use the saved key later when fetching data
  he data --type step_count --from 2024-01-01 --to 2024-01-31
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := promptAccountKey(os.Stdin, cmd.ErrOrStderr())
		if err != nil {
			return err
		}

		return loginWithKey(raw, cmd.ErrOrStderr())
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the active account key source",
	Long: strings.TrimSpace(`
Show whether an account key is currently available to the CLI.

The output reports the masked key, resolved UID, and whether the key came from
the flag, environment, or local config.
`),
	Example: strings.TrimSpace(`
  # Show the currently resolved authentication source
  he auth status

  # Check whether a one-off flag overrides config
  he auth status --account-key 0123456789abcdef0123456789abcdef
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return statusWithKey(accountKey, cmd.ErrOrStderr())
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the saved account key",
	Long: strings.TrimSpace(`
Remove the saved account key from local config.

This only clears the persisted key in ~/.config/healthexport/config.yaml. It
does not affect keys provided through HEALTHEXPORT_ACCOUNT_KEY or --account-key.
`),
	Example: strings.TrimSpace(`
  # Remove the account key saved in local config
  he auth logout

  # Confirm that no stored key remains
  he auth status
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return logoutFromConfig(cmd.ErrOrStderr())
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}

func promptAccountKey(stdin *os.File, stderr io.Writer) (string, error) {
	fmt.Fprint(stderr, "Enter your account key: ")

	fd := int(stdin.Fd())
	if term.IsTerminal(fd) {
		keyBytes, err := term.ReadPassword(fd)
		fmt.Fprintln(stderr)
		if err != nil {
			return "", fmt.Errorf("read account key: %w", err)
		}

		return string(keyBytes), nil
	}

	reader := bufio.NewReader(stdin)
	raw, err := reader.ReadString('\n')
	fmt.Fprintln(stderr)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read account key: %w", err)
	}

	return raw, nil
}

func loginWithKey(raw string, stderr io.Writer) error {
	key, err := auth.Parse(raw)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg.AccountKey = key.Raw
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(stderr, "Account key saved to %s\n", config.ConfigPath())
	fmt.Fprintf(stderr, "UID: %s\n", key.UID)

	return nil
}

func statusWithKey(flagValue string, stderr io.Writer) error {
	key, source, err := auth.Resolve(flagValue)
	if err != nil {
		if errors.Is(err, auth.ErrNoAccountKey) {
			fmt.Fprintln(stderr, "Not authenticated")
			fmt.Fprintln(stderr, "Run 'he auth login', set HEALTHEXPORT_ACCOUNT_KEY, or pass --account-key.")

			return silentExitError(err, 2)
		}

		return err
	}

	fmt.Fprintln(stderr, "Authenticated")
	fmt.Fprintf(stderr, "  Account key: %s\n", key.MaskedKey())
	fmt.Fprintf(stderr, "  UID: %s\n", key.UID)
	fmt.Fprintf(stderr, "  Source: %s\n", source)

	return nil
}

func logoutFromConfig(stderr io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.AccountKey == "" {
		fmt.Fprintln(stderr, "No account key in config")
		return nil
	}

	cfg.AccountKey = ""
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(stderr, "Account key removed from %s\n", config.ConfigPath())
	return nil
}
