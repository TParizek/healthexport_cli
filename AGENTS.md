# HealthExport CLI

HealthExport CLI is a Go command-line tool for fetching encrypted health data
from the HealthExport DataStore API and decrypting it locally on the user's
device.

## Agent Workflow

After finishing any task, always run the CLI and the unit tests before
considering the work done.

Minimum verification:

```bash
go build -o he
./he --help
./he --version
go test ./...
```

If you add or change behavior, add or update unit tests in the same task when
reasonable. Do not leave the codebase without automated coverage for the change
if a focused test can be written.

## Common Commands

Build the binary:

```bash
go build -o he
```

Run tests:

```bash
go test ./...
```

Run the full minimum verification suite:

```bash
go build -o he
./he --help
./he --version
go test ./...
```

## Project Structure

- `main.go`: entry point and version metadata wiring
- `cmd/`: Cobra commands and CLI wiring
- `cmd/root.go`: root command, global flags, env help, exit code mapping
- `cmd/data.go`: `he data` command, type resolution, local decryption flow,
  aggregation entrypoint
- `cmd/types.go`: `he types` command
- `cmd/auth.go`: `he auth login|status|logout`
- `cmd/config_cmd.go`: `he config set|get|list`
- `cmd/completion.go`: `he completion`
- `cmd/version.go`: `he version`
- `cmd/root_test.go`: baseline CLI tests for root help and version wiring
- `internal/api/`: HTTP client and API response types
- `internal/auth/`: account key parsing, UID derivation, key resolution
- `internal/config/`: config file read/write and defaults
- `internal/crypto/`: ChaCha20 decryption
- `internal/aggregator/`: client-side time-bucket aggregation
- `internal/typemap/`: type name to ID resolution and category filtering
- `internal/output/`: CSV and JSON formatters
- `testdata/`: API fixtures and crypto test vectors
- `test/README.md`: integration and vector-regeneration notes
- `requirements/`: product and task requirements, kept in-repo as
  implementation input

## Key Design Decisions

- Always fetch from the encrypted API flow and decrypt locally
- Never transmit the raw account key over the network
- Only the derived UID hash is sent to the API
- Account key resolution priority is flag > env > config
- Decryption uses ChaCha20, matching the backend implementation
- The decryption key bytes are the UTF-8 bytes of the 32-character hex string,
  not hex-decoded bytes
- Default output format is CSV; JSON is available with `--format json`
- `--raw` prints encrypted server responses without local decryption
- Human-readable status and errors belong on `stderr`; data belongs on `stdout`
- Aggregation is client-side only and valid only for compatible cumulative
  types

## Agent Notes

- Preserve the CLI contract exposed by `--help`, exit codes, and stdout/stderr
  separation
- Add or update focused tests when behavior changes
- Keep docs aligned with the actual command surface in `cmd/`
