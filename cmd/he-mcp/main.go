package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	internalmcp "github.com/TParizek/healthexport_cli/internal/mcp"
	"github.com/TParizek/healthexport_cli/internal/service"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	logger := log.New(os.Stderr, "he-mcp: ", 0)

	timeoutSeconds := parseIntEnv("HE_MCP_REQUEST_TIMEOUT_SECONDS", service.DefaultRequestTimeoutSeconds, logger)
	maxRecordsWarningThreshold := parseIntEnv("HE_MCP_MAX_RECORDS_WARNING_THRESHOLD", service.DefaultMaxRecordsWarningThreshold, logger)

	server := internalmcp.NewServer(service.Options{
		APIURL:                     sanitizeOptionalEnv(os.Getenv("HE_MCP_API_URL")),
		ConfigPath:                 sanitizeOptionalEnv(os.Getenv("HE_MCP_CONFIG_PATH")),
		RequestTimeout:             time.Duration(timeoutSeconds) * time.Second,
		MaxRecordsWarningThreshold: maxRecordsWarningThreshold,
	}, version, version, os.Stdin, os.Stdout)

	if err := server.Serve(context.Background()); err != nil {
		logger.Printf("server exited: %v (build %s %s %s)", err, version, commit, date)
		os.Exit(1)
	}
}

func parseIntEnv(key string, fallback int, logger *log.Logger) int {
	value := sanitizeOptionalEnv(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		logger.Printf("ignoring invalid %s value %q", key, value)
		return fallback
	}

	return parsed
}

func sanitizeOptionalEnv(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		return ""
	}

	switch strings.ToLower(trimmed) {
	case "null", "undefined", "<nil>", "<null>":
		return ""
	default:
		return trimmed
	}
}
