package service

import (
	"strings"
	"time"

	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/config"
)

const (
	DefaultRequestTimeoutSeconds      = 30
	DefaultMaxRecordsWarningThreshold = 5000
)

type Options struct {
	AccountKey                 string
	APIURL                     string
	ConfigPath                 string
	RequestTimeout             time.Duration
	MaxRecordsWarningThreshold int
}

func (o Options) ResolvedConfigPath() string {
	return config.ResolvePath(o.ConfigPath)
}

func (o Options) DisplayConfigPath() string {
	return config.DisplayPath(o.ResolvedConfigPath())
}

func (o Options) ResolvedAPIURL() string {
	if trimmed := strings.TrimSpace(o.APIURL); trimmed != "" {
		return trimmed
	}

	cfg, err := config.LoadFromPath(o.ResolvedConfigPath())
	if err == nil && strings.TrimSpace(cfg.APIURL) != "" {
		return strings.TrimSpace(cfg.APIURL)
	}

	return config.DefaultAPIURL
}

func (o Options) ResolvedOutputFormat(flagValue string) string {
	if trimmed := strings.TrimSpace(flagValue); trimmed != "" {
		return trimmed
	}

	cfg, err := config.LoadFromPath(o.ResolvedConfigPath())
	if err == nil && strings.TrimSpace(cfg.Format) != "" {
		return strings.TrimSpace(cfg.Format)
	}

	return config.DefaultFormat
}

func (o Options) EffectiveRequestTimeout() time.Duration {
	if o.RequestTimeout > 0 {
		return o.RequestTimeout
	}

	return DefaultRequestTimeoutSeconds * time.Second
}

func (o Options) EffectiveMaxRecordsWarningThreshold() int {
	if o.MaxRecordsWarningThreshold > 0 {
		return o.MaxRecordsWarningThreshold
	}

	return DefaultMaxRecordsWarningThreshold
}

func (o Options) APIClient() *api.Client {
	return api.NewClientWithTimeout(o.ResolvedAPIURL(), o.EffectiveRequestTimeout())
}
