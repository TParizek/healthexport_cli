package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/aggregator"
	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/TParizek/healthexport_cli/internal/service"
	"github.com/TParizek/healthexport_cli/internal/typemap"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	accountKey string
	apiURL     string

	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const (
	envHelpAnnotation = "healthexport_env"
	usageTemplate     = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{with .Annotations}}{{with index . "healthexport_env"}}

Environment Variables:
{{.}}{{end}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
)

var rootCmd = &cobra.Command{
	Use:   "he",
	Short: "Read and decrypt your health data",
	Long: strings.TrimSpace(`
HealthExport CLI - read and decrypt your health data.

Records are always fetched encrypted and decrypted locally.
Your account key never leaves this machine.
`),
	Annotations: map[string]string{
		envHelpAnnotation: "  HEALTHEXPORT_ACCOUNT_KEY   Account key (overrides config, overridden by --account-key flag)",
	},
	Version: versionString(),
}

func init() {
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.InitDefaultVersionFlag()
	_ = rootCmd.Flags().MarkHidden("version")
	rootCmd.PersistentFlags().StringVar(&accountKey, "account-key", "", "Account key (overrides env and config)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "API base URL (overrides config)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return initConfig()
	}

	_ = viper.BindPFlag("account_key", rootCmd.PersistentFlags().Lookup("account-key"))
	_ = viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
}

// Execute runs the root command and exits with code 1 on failure.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if shouldPrintError(err) {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}

		os.Exit(exitCodeForError(err))
	}
}

// SetVersionInfo sets build-time version metadata.
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
	rootCmd.Version = versionString()
	api.SetUserAgentVersion(v)
}

func initConfig() error {
	viper.SetEnvPrefix("HEALTHEXPORT")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetConfigFile(config.ConfigPath())
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read config file: %w", err)
		}
	}

	return nil
}

func versionString() string {
	return fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

type cliExitError struct {
	err   error
	code  int
	print bool
}

func (e *cliExitError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}

	return e.err.Error()
}

func (e *cliExitError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.err
}

func silentExitError(err error, code int) error {
	return &cliExitError{
		err:   err,
		code:  code,
		print: false,
	}
}

func exitError(err error, code int) error {
	return &cliExitError{
		err:   err,
		code:  code,
		print: true,
	}
}

func exitCodeForError(err error) int {
	var cliErr *cliExitError
	if errors.As(err, &cliErr) {
		return cliErr.code
	}

	var apiErr *api.APIError

	switch {
	case errors.Is(err, auth.ErrNoAccountKey):
		return 2
	case errors.Is(err, auth.ErrInvalidKeyFormat):
		return 4
	case errors.Is(err, service.ErrInvalidInput):
		return 4
	case errors.As(err, &apiErr):
		return 3
	case errors.Is(err, typemap.ErrUnknownType):
		return 4
	case errors.Is(err, aggregator.ErrNotAggregatable):
		return 4
	default:
		return 1
	}
}

func shouldPrintError(err error) bool {
	var cliErr *cliExitError
	if errors.As(err, &cliErr) {
		return cliErr.print && cliErr.err != nil
	}

	return err != nil
}
