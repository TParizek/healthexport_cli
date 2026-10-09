# HealthExport Local MCP Extension

This guide covers the local macOS extension. Compare
[hosted and local MCP](../README.md#mcp) to choose where the server runs and
where records are decrypted. Hosted MCP uses a server URL and Apple sign-in;
the local extension below uses CLI credentials and local decryption.

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
- `list_workouts`
- `fetch_workout_detail`
- `fetch_workout_series`

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

## Workout tools

These tools read only previously uploaded workout details. They do not trigger sync or include summary-only workouts. Distinct workout IDs remain distinct.

1. `list_workouts`: `start` and `end` are inclusive RFC3339 timestamps, at most millisecond precision and 366 days apart. Returns manifests only; `limit` is 1–10 (default 10), `offset` defaults to 0. Follow `nextOffset` until null.
2. `fetch_workout_detail`: pass `workoutId` and `revision` from the listing. Returns the manifest plus events and recorded `splitUnit` (`km` default, or `mi`). `limit` is 1–4 source pages per section (default 1). Each section has its own `nextOffset`; repeat using each unfinished section's offset. No numeric streams are loaded. Split distances remain meters for both split sets.
3. `fetch_workout_series`: pass `workoutId`, `revision`, and one `section`: `heartRate`, `distance`, `speed`, `elevation`, `power`, or `cadence`. `limit` is 1–8 source pages (default 1), each with at most 256 stored points. Follow `nextOffset`. Optional `unit` converts values and extrema to a compatible unit (`km`, `mi`, `ft`, `km/h`, `mph`) or uses the canonical unit.

For example, list a week's workouts, select the relevant ID/revision/source, then fetch only its heart-rate stream. Use the manifest's availability to distinguish missing data from zero values. Offsets, durations and coverage are seconds; `breakBefore` marks gaps. Preserve aggregation interval/method/source and point `kind`, `min`, `max`, and `count` when interpreting the data. Stored means do not reconstruct original HealthKit samples or raw peaks.

Revision conflicts return an API error: list again and restart the read, without mixing uploads. MCP results remain bounded; the CLI's `he workouts export` intentionally follows all pages to produce the complete website-compatible JSON document. FIT/TCX/GPX export is not supported.
