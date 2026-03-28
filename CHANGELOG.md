# Changelog

All notable changes to this project will be documented in this file.

## [1.1.0] - 2026-03-28

- Adds a local macOS MCP extension for Claude Desktop and Claude Cowork.
- Adds `health_export_status`, `list_health_types`, and `fetch_health_data` MCP tools.
- Adds `he mcp status` for local MCP diagnostics.
- Refactors shared fetch, type resolution, aggregation, and status logic.
- Fixes MCP input validation and mixed aggregation handling.
- Rounds aggregated `count` values to remove float noise.
- Adds MCP packaging, icon, companion skill, and release assets.
- Updates README and MCP docs for release downloads and setup.

## [1.0.0] - 2026-03-24

- Initial release of HealthExport CLI.
- Adds command-line access to HealthExport records with local decryption.
- Includes `auth`, `data`, `types`, `config`, `completion`, and `version` commands.
- Supports CSV and JSON output formats, plus client-side aggregation for compatible data types.
