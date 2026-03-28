package service

import (
	"errors"
	"fmt"

	"github.com/TParizek/healthexport_cli/internal/auth"
)

type Status struct {
	HEVersion     string `json:"he_version"`
	Authenticated bool   `json:"authenticated"`
	AuthSource    string `json:"auth_source"`
	ConfigPath    string `json:"config_path"`
	APIURL        string `json:"api_url"`
}

type MCPStatus struct {
	ServerVersion        string `json:"server_version"`
	CompatibleCLIVersion string `json:"compatible_cli_version"`
	HEVersion            string `json:"he_version"`
	Authenticated        bool   `json:"authenticated"`
	AuthSource           string `json:"auth_source"`
	ConfigPath           string `json:"config_path"`
	APIURL               string `json:"api_url"`
}

func GetStatus(opts Options, heVersion string) (*Status, error) {
	status := &Status{
		HEVersion:  heVersion,
		ConfigPath: opts.DisplayConfigPath(),
		APIURL:     opts.ResolvedAPIURL(),
	}

	_, source, err := auth.ResolveWithConfigPath(opts.AccountKey, opts.ConfigPath)
	switch {
	case err == nil:
		status.Authenticated = true
		status.AuthSource = source
		return status, nil
	case errors.Is(err, auth.ErrNoAccountKey):
		return status, nil
	default:
		return nil, fmt.Errorf("%w: %w", ErrConfig, err)
	}
}

func GetMCPStatus(opts Options, heVersion, serverVersion, compatibleCLIVersion string) (*MCPStatus, error) {
	status, err := GetStatus(opts, heVersion)
	if err != nil {
		return nil, err
	}

	return &MCPStatus{
		ServerVersion:        serverVersion,
		CompatibleCLIVersion: compatibleCLIVersion,
		HEVersion:            status.HEVersion,
		Authenticated:        status.Authenticated,
		AuthSource:           status.AuthSource,
		ConfigPath:           status.ConfigPath,
		APIURL:               status.APIURL,
	}, nil
}
