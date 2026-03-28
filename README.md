# HealthExport CLI (he)

[![Release](https://github.com/TParizek/healthexport_cli/actions/workflows/release.yml/badge.svg)](https://github.com/TParizek/healthexport_cli/actions/workflows/release.yml)
[![Latest Release](https://img.shields.io/github/v/tag/TParizek/healthexport_cli?label=release&sort=semver)](https://github.com/TParizek/healthexport_cli/releases)
[![License: MIT](https://img.shields.io/github/license/TParizek/healthexport_cli?label=license)](https://github.com/TParizek/healthexport_cli/blob/main/LICENSE)

Read and decrypt your health data from the command line.

[HealthExport](https://healthexport.app) is an iPhone app for exporting health
data from your iPhone and viewing it in formats such as CSV.

[HealthExport Remote](https://remote.healthexport.app) is an additional service
that lets users access the same iPhone health data remotely in a browser, with
background sync and end-to-end encryption.

This repository contains the CLI tool for accessing those same records from the
terminal. HealthExport CLI fetches encrypted health records from the
HealthExport DataStore API and decrypts them locally. Your account key never
leaves your machine.

## Download

Start at [GitHub Releases](https://github.com/TParizek/healthexport_cli/releases).

Most people should download from the latest release page:
- `he_<version>_darwin_arm64.tar.gz` for Apple Silicon Macs
- `he_<version>_darwin_amd64.tar.gz` for Intel Macs
- `he_<version>_linux_arm64.tar.gz` or `he_<version>_linux_amd64.tar.gz` for Linux

If you want to use Claude Desktop or Claude Cowork on macOS, also download:
- `health-export_<version>_darwin_arm64.mcpb` for Apple Silicon Macs
- `health-export_<version>_darwin_amd64.mcpb` for Intel Macs

Homebrew is also available for the CLI:

```bash
brew tap TParizek/healthexport_tap https://github.com/TParizek/healthexport_tap
brew install TParizek/healthexport_tap/he
```

## Quick Start

### CLI setup

1. Download the right `he` file from [GitHub Releases](https://github.com/TParizek/healthexport_cli/releases).
2. Extract it and place `he` somewhere on your `PATH`.
3. Open Terminal.
4. Run `he auth login`.
5. Paste your account key from https://remote.healthexport.app/settings/sharing
6. Test it with `he types`.

### Claude Desktop setup on macOS

1. Install `he` first and run `he auth login`.
2. Download the matching `health-export_<version>_darwin_<arch>.mcpb` file from [GitHub Releases](https://github.com/TParizek/healthexport_cli/releases).
3. Open Claude Desktop.
4. Go to `Settings > Extensions`.
5. Choose `Advanced settings > Install Extension...`.
6. Select the `.mcpb` file.
7. Ask Claude to run `health_export_status`.

Full Claude setup and troubleshooting instructions are in [docs/mcp.md](docs/mcp.md).

## Common Commands

```bash
# Save your account key
he auth login

# List available data types
he types

# Fetch step count for a time range
he data --type step_count --from 2024-01-01 --to 2024-01-07

# Check Claude Desktop extension readiness
he mcp status --format json
```

Run `he --help` or `he <command> --help` for more details.

## Commands

### `he data`

Fetch encrypted health records, decrypt them locally, and print structured
output.

Key flags:
- `--type`, `-t`: data type name or numeric ID; repeat for multiple types
- `--from`, `-f`: start date in `YYYY-MM-DD` or RFC3339
- `--to`, `-T`: end date in `YYYY-MM-DD` or RFC3339
- `--format`: `csv` or `json`
- `--aggregate`, `-a`: `day`, `week`, `month`, or `year` for compatible types
- `--raw`: print encrypted API payloads as JSON without local decryption

### `he types`

List available health data types, including canonical names, numeric IDs, and
which types support aggregation.

Key flags:
- `--format`: `csv` or `json`
- `--category`: `aggregated`, `record`, or `workout`

### `he auth`

Manage your local account key:
- `he auth login`: prompt for and save the account key to local config
- `he auth status`: show the active auth source, masked key, and derived UID
- `he auth logout`: remove the stored key from config

### `he config`

Inspect and update local CLI configuration:
- `he config set <key> <value>`
- `he config get <key>`
- `he config list`

Supported keys: `format`, `api_url`, `account_key`

### `he mcp`

Inspect local MCP integration status for the Claude Desktop extension:
- `he mcp status`: human-readable host readiness summary
- `he mcp status --format json`: stable machine-readable diagnostics

### Other Commands

- `he completion bash|zsh|fish|powershell`: generate shell completions
- `he version` or `he --version`: print build metadata

## Authentication

The CLI supports three ways to provide your account key:

You can view your account key in HealthExport Remote:
https://remote.healthexport.app/settings/sharing

1. Config file (recommended): `he auth login`
2. Environment variable: `export HEALTHEXPORT_ACCOUNT_KEY=...`
3. Flag: `--account-key "..."`

Resolution priority is: `--account-key` > `HEALTHEXPORT_ACCOUNT_KEY` >
config file.

The config file is stored at `~/.config/healthexport/config.yaml` on typical
macOS and Linux setups, or under `XDG_CONFIG_HOME` when set.

## Output Formats

- `csv` (default): good for spreadsheets, `awk`, `cut`, and shell pipelines
- `json`: good for `jq`, scripts, and agent/tool integration
- `--raw`: prints encrypted server responses as JSON for inspection/debugging

Human-oriented messages go to `stderr`. Structured data goes to `stdout`.

## Aggregation

For cumulative data types such as steps, distance, and calories, the CLI can
aggregate records client-side after decryption:

```bash
he data --type step_count --from 2024-01-01 --to 2024-12-31 --aggregate month
```

Supported periods: `day`, `week`, `month`, `year`

## Agent Integration

This CLI is designed to work well with coding agents and other automation:

- Structured output on `stdout` (`csv` or `json`)
- Human messages on `stderr`
- Predictable exit codes: `0` success, `2` no auth, `3` API error, `4` bad input
- Comprehensive `--help` on every command
- Shell completions via `he completion ...`
- One-off auth via `HEALTHEXPORT_ACCOUNT_KEY=xxx he data ...`

## Claude Desktop / Cowork

HealthExport includes a local macOS extension for Claude Desktop and Claude
Cowork. Download the `.mcpb` file from [GitHub Releases](https://github.com/TParizek/healthexport_cli/releases), install `he`, run `he auth login`, then install the extension in Claude Desktop.

The extension exposes three read-only tools:
- `health_export_status`
- `list_health_types`
- `fetch_health_data`

Full setup and troubleshooting instructions are in [docs/mcp.md](docs/mcp.md).

## Security

- Your account key never leaves your machine
- The CLI always fetches from the encrypted API endpoint
- Only the derived UID hash is sent to the API
- Decryption uses ChaCha20 locally in the CLI process
- `--raw` lets you inspect the encrypted payload returned by the server
- Config directories are written with `0700` permissions and the config file
  with `0600`

## Development

Minimum local verification:

```bash
go build -o he
go build -o he-mcp ./cmd/he-mcp
./he --help
./he --version
go test ./...
```

Backend-derived crypto vectors, manual end-to-end verification notes, and test
asset details live in `test/README.md`.

## License

MIT
