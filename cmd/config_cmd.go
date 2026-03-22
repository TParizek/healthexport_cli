package cmd

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local CLI configuration values",
	Long: strings.TrimSpace(`
Inspect and update local CLI configuration stored on this machine.

Use the subcommands below to set defaults, read individual values, or list the
resolved configuration currently on disk.
`),
	Example: strings.TrimSpace(`
  # Show all stored configuration values
  he config list

  # Set the default output format
  he config set format json

  # Read a single configuration value
  he config get api_url
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a local configuration value",
	Long: strings.TrimSpace(`
Set a configuration value.

Valid keys:
  format       Default output format (csv, json)
  api_url      API base URL
  account_key  Account key (prefer 'he auth login' instead)
`),
	Example: strings.TrimSpace(`
  # Set JSON as the default output format
  he config set format json

  # Point the CLI at a custom API base URL
  he config set api_url https://custom.example.com/api/v2
`),
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigSet(args[0], args[1], cmd.ErrOrStderr())
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a resolved configuration value",
	Long: strings.TrimSpace(`
Get a single configuration value by key.

If a value is not stored in config, the command prints the built-in default
for that key instead.
`),
	Example: strings.TrimSpace(`
  # Print the configured API base URL
  he config get api_url

  # Print the resolved default output format
  he config get format
`),
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigGet(args[0], cmd.OutOrStdout())
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all resolved configuration values",
	Long: strings.TrimSpace(`
List every supported configuration key and its current value.

Stored account keys are masked in the output so you can inspect the config
without exposing the full secret.
`),
	Example: strings.TrimSpace(`
  # List all configuration values
  he config list

  # Review configuration before running data export commands
  he config list
`),
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigList(cmd.OutOrStdout())
	},
}

func init() {
	configCmd.AddCommand(configSetCmd, configGetCmd, configListCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(key, value string, stderr io.Writer) error {
	normalizedKey, err := validateConfigKey(key)
	if err != nil {
		return exitError(err, 4)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := cfg.SetField(normalizedKey, value); err != nil {
		return exitError(err, 4)
	}

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(stderr, "Config updated: %s = %s\n", normalizedKey, value)
	return nil
}

func runConfigGet(key string, stdout io.Writer) error {
	normalizedKey, err := validateConfigKey(key)
	if err != nil {
		return exitError(err, 4)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	value, err := resolvedConfigValue(cfg, normalizedKey)
	if err != nil {
		return exitError(err, 4)
	}

	fmt.Fprintln(stdout, value)
	return nil
}

func runConfigList(stdout io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	for _, key := range config.ValidKeys() {
		value, err := resolvedConfigValue(cfg, key)
		if err != nil {
			return exitError(err, 4)
		}

		if key == "account_key" {
			value = maskAccountKey(value)
		}

		fmt.Fprintf(stdout, "%s=%s\n", key, value)
	}

	return nil
}

func validateConfigKey(key string) (string, error) {
	normalized := strings.TrimSpace(key)
	if slices.Contains(config.ValidKeys(), normalized) {
		return normalized, nil
	}

	return "", fmt.Errorf("unknown config key %q", key)
}

func resolvedConfigValue(cfg *config.Config, key string) (string, error) {
	value, err := cfg.GetField(key)
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(value) != "" {
		return value, nil
	}

	switch key {
	case "format":
		return config.DefaultFormat, nil
	case "api_url":
		return config.DefaultAPIURL, nil
	case "account_key":
		return "", nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func maskAccountKey(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	parsed, err := auth.Parse(raw)
	if err != nil {
		return raw
	}

	return parsed.MaskedKey()
}
