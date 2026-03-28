package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	DefaultAPIURL = "https://remoteapi.healthexport.app/api/v2"
	DefaultFormat = "csv"
)

type Config struct {
	AccountKey string `yaml:"account_key,omitempty"`
	Format     string `yaml:"format,omitempty"`
	APIURL     string `yaml:"api_url,omitempty"`
}

func ConfigDir() string {
	return filepath.Join(configBaseDir(), "healthexport")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

func Load() (*Config, error) {
	return LoadFromPath(ConfigPath())
}

func LoadFromPath(path string) (*Config, error) {
	resolvedPath := ResolvePath(path)

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}

		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	return c.SaveToPath(ConfigPath())
}

func (c *Config) SaveToPath(path string) error {
	if c == nil {
		return errors.New("config is nil")
	}

	resolvedPath := ResolvePath(path)
	configDir := filepath.Dir(resolvedPath)

	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.Chmod(configDir, 0o700); err != nil {
		return fmt.Errorf("chmod config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(resolvedPath, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	if err := os.Chmod(resolvedPath, 0o600); err != nil {
		return fmt.Errorf("chmod config file: %w", err)
	}

	return nil
}

func ResolvePath(override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}

	return ConfigPath()
}

func DisplayPath(path string) string {
	resolvedPath := ResolvePath(path)
	if resolvedPath == ConfigPath() {
		return "~/.config/healthexport/config.yaml"
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return resolvedPath
	}

	if resolvedPath == home {
		return "~"
	}

	prefix := home + string(os.PathSeparator)
	if strings.HasPrefix(resolvedPath, prefix) {
		return "~" + string(os.PathSeparator) + strings.TrimPrefix(resolvedPath, prefix)
	}

	return resolvedPath
}

func (c *Config) SetField(key, value string) error {
	if c == nil {
		return errors.New("config is nil")
	}

	switch key {
	case "account_key":
		c.AccountKey = value
	case "format":
		if value != "csv" && value != "json" {
			return fmt.Errorf("invalid format %q: must be csv or json", value)
		}

		c.Format = value
	case "api_url":
		if err := validateAPIURL(value); err != nil {
			return err
		}

		c.APIURL = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}

	return nil
}

func (c *Config) GetField(key string) (string, error) {
	if c == nil {
		return "", errors.New("config is nil")
	}

	switch key {
	case "account_key":
		return c.AccountKey, nil
	case "format":
		return c.Format, nil
	case "api_url":
		return c.APIURL, nil
	default:
		return "", fmt.Errorf("unknown config key %q", key)
	}
}

func ValidKeys() []string {
	return []string{"account_key", "format", "api_url"}
}

func configBaseDir() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return xdgConfigHome
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, ".config")
		}
	}

	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return dir
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".config")
	}

	return ".config"
}

func validateAPIURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid api_url %q: %w", value, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid api_url %q: must start with http:// or https://", value)
	}

	if parsed.Host == "" {
		return fmt.Errorf("invalid api_url %q: host is required", value)
	}

	return nil
}
