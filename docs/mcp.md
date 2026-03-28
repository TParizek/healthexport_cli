# HealthExport MCP Extension

HealthExport ships a local Model Context Protocol server for Claude Desktop and
Claude Cowork as a macOS-only `.mcpb` extension.

The extension does not proxy health data through a remote service. It reuses
the same local HealthExport config as the `he` CLI, fetches encrypted records
from the HealthExport API, and decrypts them locally on the host machine.

## What to download

Go to [GitHub Releases](https://github.com/TParizek/healthexport_cli/releases) and download:

1. The `he` CLI for your Mac.
2. The matching `.mcpb` file for your Mac:
   `health-export_<version>_darwin_arm64.mcpb` for Apple Silicon or
   `health-export_<version>_darwin_amd64.mcpb` for Intel.

## Step-by-step setup

1. Install `he` on your Mac.
2. Open Terminal.
3. Run `he auth login`.
4. Paste your account key from https://remote.healthexport.app/settings/sharing
5. Run `he mcp status --format json`.
6. Open Claude Desktop.
7. Go to `Settings > Extensions`.
8. Choose `Advanced settings > Install Extension...`.
9. Select the `.mcpb` file you downloaded.
10. Leave the optional settings empty unless you need a custom config path or API URL.
11. Enable the extension.
12. Ask Claude to run `health_export_status`.

## Install into Claude Desktop

The extension uses the same local HealthExport config as the `he` CLI. If
`he auth login` worked on your Mac, the extension should normally work without
extra setup.

## Available tools

- `health_export_status`
- `list_health_types`
- `fetch_health_data`

All tools are read-only. The server never returns the raw account key.

## Recommended first-run checks

1. Ask Claude to run `health_export_status`.
2. Ask Claude to run `list_health_types`.
3. Ask Claude to run `fetch_health_data` for a short range such as 7 days of
   `step_count`.

## Troubleshooting

### Missing `he`

The bundled extension runs its own `he-mcp` server binary and does not shell
out to `he`, but the supported setup still assumes `he` is installed so users
can authenticate with `he auth login` and inspect readiness with
`he mcp status`.

### Missing auth

If `health_export_status` reports `authenticated: false`, run:

```bash
he auth login
```

The extension reads the same config file as the CLI by default.

### Custom config path

If your HealthExport config is not stored at the default path, set the optional
`configPath` extension setting to the full path of the config file.

### Permission issues

If Claude Desktop cannot read the configured file path, re-open the extension
settings and confirm the `configPath` points to a readable location on the host
machine.

### API override issues

If you set `apiURL`, verify that the endpoint is reachable and matches the
HealthExport API surface expected by the CLI.

## Development build

If you are working from source, you can still build the extension locally:

```bash
./scripts/build_mcpb.sh
```
