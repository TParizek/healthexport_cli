# HealthExport CLI (he)

[![Release](https://github.com/TParizek/healthexport_cli/actions/workflows/release.yml/badge.svg)](https://github.com/TParizek/healthexport_cli/actions/workflows/release.yml) [![Latest Release](https://img.shields.io/github/v/tag/TParizek/healthexport_cli?label=release&sort=semver)](https://github.com/TParizek/healthexport_cli/releases) [![License: MIT](https://img.shields.io/github/license/TParizek/healthexport_cli?label=license)](https://github.com/TParizek/healthexport_cli/blob/main/LICENSE)

Read and decrypt your health data from the command line.

[HealthExport](https://healthexport.app) is an iPhone app for exporting health data from your iPhone and viewing it in formats such as CSV.

[HealthExport Remote](https://remote.healthexport.app) is an additional service that lets users access the same iPhone health data remotely in a browser, with background uploads and encrypted storage. The browser decrypts records on your device; the optional hosted MCP connection uses consented server-memory decryption to share requested results with your AI service.

This repository contains the CLI tool for accessing those same records from the terminal. HealthExport CLI fetches encrypted health records from the HealthExport DataStore API and decrypts them locally. In this CLI workflow, your account key never leaves your machine.

## CLI setup

Use `he` with coding agents such as Codex and Claude Code, or your own scripts. The CLI fetches encrypted Remote records, decrypts them on your computer, and returns CSV or JSON. When you share its output with an AI assistant, that client and its provider receive those records.

### Download

Download a CLI archive from [GitHub Releases](https://github.com/TParizek/healthexport_cli/releases):

- `he_<version>_darwin_arm64.tar.gz` for Apple Silicon Macs
- `he_<version>_darwin_amd64.tar.gz` for Intel Macs
- `he_<version>_linux_arm64.tar.gz` or `he_<version>_linux_amd64.tar.gz` for Linux

Extract the archive and place `he` somewhere on your `PATH`.

Alternatively, install with Homebrew:

```bash
brew tap TParizek/healthexport_tap https://github.com/TParizek/healthexport_tap
brew install TParizek/healthexport_tap/he
```

### Quick start

1. Open Terminal and run `he auth login`.
2. Paste your account key from [Remote sharing settings](https://remote.healthexport.app/settings/sharing) when prompted.
3. Run `he types` to list supported data types.
4. Fetch a date range you have uploaded:

```bash
he data --type step_count --from 2024-01-01 --to 2024-01-07 --format json
```

Replace the example dates with your own. Hosted MCP sign-in does not authenticate the CLI; `he auth login` saves credentials for local use. Treat your account key like a password. Unlike a hosted MCP grant, a shared account key cannot currently be revoked or rotated.

## Common Commands

```bash
# Save your account key
he auth login

# List available data types
he types

# Fetch step count for a time range
he data --type step_count --from 2024-01-01 --to 2024-01-07
```

Run `he --help` or `he <command> --help` for more details.

## Commands

### `he data`

Fetch encrypted health records, decrypt them locally, and print structured output.

Key flags:
- `--type`, `-t`: data type name or numeric ID; repeat for multiple types
- `--from`, `-f`: start date in `YYYY-MM-DD` or RFC3339
- `--to`, `-T`: end date in `YYYY-MM-DD` or RFC3339
- `--format`: `csv` or `json`
- `--aggregate`, `-a`: `day`, `week`, `month`, or `year` for compatible types
- `--raw`: print encrypted API payloads as JSON without local decryption

### `he types`

List available health data types, including canonical names, numeric IDs, and which types support aggregation.

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

Inspect local MCP readiness (this does not check the hosted MCP connection):
- `he mcp status`: human-readable host readiness summary
- `he mcp status --format json`: stable machine-readable diagnostics

Local extension setup and troubleshooting are documented in [docs/mcp.md](docs/mcp.md).

### Other Commands

- `he completion bash|zsh|fish|powershell`: generate shell completions
- `he version` or `he --version`: print build metadata

## Authentication

The CLI supports three ways to provide your account key:

You can view your account key in HealthExport Remote: https://remote.healthexport.app/settings/sharing

1. Config file (recommended): `he auth login`
2. Environment variable: `export HEALTHEXPORT_ACCOUNT_KEY=...`
3. Flag: `--account-key "..."`

Resolution priority is: `--account-key` > `HEALTHEXPORT_ACCOUNT_KEY` > config file.

The config file is stored at `~/.config/healthexport/config.yaml` on typical macOS and Linux setups, or under `XDG_CONFIG_HOME` when set.

## Output Formats

- `csv` (default): good for spreadsheets, `awk`, `cut`, and shell pipelines
- `json`: good for `jq`, scripts, and agent/tool integration
- `--raw`: prints encrypted server responses as JSON for inspection/debugging

Human-oriented messages go to `stderr`. Structured data goes to `stdout`.

## Aggregation

For cumulative data types such as steps, distance, and calories, the CLI can aggregate records client-side after decryption:

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

## CLI security

These properties describe this CLI and its local extension, not hosted MCP. When used with an AI assistant, requested results are shared with that client.

- Your account key never leaves your machine
- The CLI always fetches from the encrypted API endpoint
- Only the derived UID hash is sent to the API
- Decryption uses ChaCha20 locally in the CLI process
- `--raw` lets you inspect the encrypted payload returned by the server
- Config directories are written with `0700` permissions and the config file with `0600`

## MCP

HealthExport offers **two MCP connections**: a hosted service and a local server. Both let compatible AI clients read health records already uploaded to HealthExport Remote. Neither connects directly to your watch or HealthKit, changes your health records, or triggers an iPhone upload.

Enable Remote on your iPhone, grant Apple Health access, and finish the initial upload before using either option. Background upload timing depends on iOS. Your AI provider's account requirements also apply.

| | Hosted MCP | Local MCP |
| --- | --- | --- |
| Best fit | Connect with a URL and Apple sign-in | Run the server and decrypt records on your computer |
| Setup | Add the server URL in your client; no local installation | Install the CLI, authenticate, and configure the local server or macOS extension |
| Connection | Remote MCP over Streamable HTTP | Local `he-mcp` process over standard input/output (stdio) |
| Authentication | Sign in with Apple and approve read-only access | Account key saved locally with `he auth login` |
| Decryption | Requested records are decrypted in hosted server memory | Records are decrypted on your computer |
| Tools | Status, type catalog, individual records, and a separate aggregation tool | Status, type catalog, and record fetching with an optional aggregation parameter |
| Request size | Up to 7 days, 4 health types, and 2,000 input records per request | Does not use those hosted limits; large responses produce warnings |
| Clients | Claude, the Codex app, or another compatible remote MCP client | Clients that can launch a local stdio server; a macOS `.mcpb` package is provided for Claude Desktop |

**Choose hosted MCP** for the simplest setup. **Choose local MCP** if you want records decrypted on your computer and are comfortable installing local tools. With either option, results passed to an AI client may be sent to its provider; local decryption does not mean your AI conversation stays on your device.

<details>
<summary><strong>Hosted MCP: setup and privacy</strong></summary>

Connect Claude, the Codex app, or another compatible remote MCP client with a server URL and Apple sign-in. No CLI installation or account key is needed. Your AI provider's account requirements also apply.

```text
https://mcp.healthexport.app/mcp
```

1. Add the URL above as a remote MCP server or custom connector in your AI client's settings. Your client needs Streamable HTTP and browser-based OAuth sign-in support.
2. Sign in with the **same Apple account** you use for HealthExport and approve read-only access.
3. Enable the connection in your AI client and try the prompt below.

**Try asking:**

```text
Show my uploaded step totals by day for the last three days. Flag any missing records.
```

See the [MCP setup guide](https://healthexport.app/apple-watch-mcp.html#setup) for Codex app, Claude and other client instructions. Claude Desktop has been tested with HealthExport; other clients have not yet been verified end to end.

Hosted MCP reads **previously uploaded records**. It cannot access your watch or HealthKit directly, trigger an iPhone sync, or modify health records. Background upload timing depends on iOS. Queries support up to **7 days, 4 health types and 2,000 input records** per request; oversized requests must be narrowed. Only eligible numeric types can be summed.

The hosted service exposes four read-only tools: `health_export_status`, `list_health_types`, `fetch_health_data`, and `fetch_aggregated_health_data`. Status does not verify data freshness, and the catalog does not prove that your account contains every listed type.

**Privacy:** hosted MCP decrypts requested records in server memory and sends results to your AI client and its provider. Customer decryption keys are not stored in the MCP database; decryption material is present in server memory during use. Review your AI provider's privacy settings before connecting.

</details>

<details>
<summary><strong>Local MCP: setup and privacy</strong></summary>

The local `he-mcp` server uses the same configuration and account key as the CLI. It fetches encrypted records from Remote and decrypts them in the local process. It requires an internet connection to fetch uploaded records.

For the packaged macOS extension:

1. [Install the CLI](#cli-setup) and run `he auth login`.
2. Download the matching `.mcpb` file from [GitHub Releases](https://github.com/TParizek/healthexport_cli/releases): `health-export_<version>_darwin_arm64.mcpb` for Apple Silicon or `health-export_<version>_darwin_amd64.mcpb` for Intel Macs.
3. Install the package in Claude Desktop following the [local extension guide](docs/mcp.md).
4. Ask the connected client to run `health_export_status`, then `list_health_types` before requesting a short date range.

For another client that supports local stdio servers, first [install and authenticate the CLI](#cli-setup). Then, from a checkout of this repository with Go installed, build `he-mcp` and configure the client to launch the resulting executable:

```bash
go build -o bin/he-mcp ./cmd/he-mcp
```

Use the executable's absolute path in your client's MCP configuration. It reads the same local config as `he`; set `HE_MCP_CONFIG_PATH` in the server's environment if you use a custom config path. Other client configurations have not been verified end to end with HealthExport.

The local server exposes three read-only tools: `health_export_status`, `list_health_types`, and `fetch_health_data`. For eligible cumulative types, `fetch_health_data` accepts an `aggregate` parameter with `day`, `week`, `month`, or `year`. Hosted MCP's per-request limits above describe the hosted service, not this local server; keep local requests focused to avoid large responses.

**Privacy:** your account key stays on your computer, and decryption happens locally. Requested results still go to the connected AI client. Treat your account key like a password; unlike a hosted MCP grant, a shared account key cannot currently be revoked or rotated. Hosted Apple sign-in does not authenticate the local server.

See [docs/mcp.md](docs/mcp.md) for extension setup and troubleshooting.

</details>

## Development

Minimum local verification:

```bash
go build -o he
go build -o he-mcp ./cmd/he-mcp
./he --help
./he --version
go test ./...
```

Backend-derived crypto vectors, manual end-to-end verification notes, and test asset details live in `test/README.md`.

## License

MIT
