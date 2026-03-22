package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/TParizek/healthexport_cli/internal/config"
)

const EnvKeyName = "HEALTHEXPORT_ACCOUNT_KEY"

var ErrNoAccountKey = errors.New("no account key configured")

func Resolve(flagValue string) (*AccountKey, string, error) {
	if flagValue != "" {
		return parseFromSource(flagValue, "--account-key flag")
	}

	if envValue := os.Getenv(EnvKeyName); envValue != "" {
		return parseFromSource(envValue, EnvKeyName+" env var")
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, "", fmt.Errorf("load config: %w", err)
	}

	if cfg.AccountKey != "" {
		return parseFromSource(cfg.AccountKey, "~/.config/healthexport/config.yaml")
	}

	return nil, "", ErrNoAccountKey
}

func parseFromSource(raw, source string) (*AccountKey, string, error) {
	key, err := Parse(raw)
	if err != nil {
		return nil, "", fmt.Errorf("parse account key from %s: %w", source, err)
	}

	return key, source, nil
}
