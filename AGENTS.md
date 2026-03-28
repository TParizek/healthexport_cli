# HealthExport CLI

HealthExport CLI is a Go command-line tool for fetching encrypted health data
from the HealthExport DataStore API and decrypting it locally on the user's
device.

## Agent Workflow

After finishing any task, always run the CLI, vet, lint, and the unit tests
before considering the work done.

Minimum verification:

```bash
go vet ./...
./.bin/golangci-lint run
go build -o he
./he --help
./he --version
go test ./...
```

If you add or change behavior, add or update unit tests in the same task when
reasonable. Do not leave the codebase without automated coverage for the change
if a focused test can be written.

If you change `.github/workflows/release.yml`, `.goreleaser.yaml`, or
`scripts/build_mcpb.sh`, also test the release pipeline locally before
considering the work done. At minimum:

```bash
./.bin/goreleaser release --snapshot --clean --verbose
GOCACHE="$PWD/.cache/go-build" GOMODCACHE="$PWD/.cache/gomod" HE_MCPB_VERSION=v1.0.0-local HE_MCPB_COMMIT=$(git rev-parse HEAD) HE_MCPB_DATE=$(git log -1 --format=%cI) GOARCH=arm64 ./scripts/build_mcpb.sh dist/health-export_1.0.0-local_darwin_arm64.mcpb
GOCACHE="$PWD/.cache/go-build" GOMODCACHE="$PWD/.cache/gomod" HE_MCPB_VERSION=v1.0.0-local HE_MCPB_COMMIT=$(git rev-parse HEAD) HE_MCPB_DATE=$(git log -1 --format=%cI) GOARCH=amd64 ./scripts/build_mcpb.sh dist/health-export_1.0.0-local_darwin_amd64.mcpb
```

## Common Commands

Build the binary:

```bash
go build -o he
```

Run tests:

```bash
go test ./...
```

Install the CI-matching linter locally:

```bash
mkdir -p .bin .cache/go-build .cache/gomod
GOBIN="$PWD/.bin" GOCACHE="$PWD/.cache/go-build" GOMODCACHE="$PWD/.cache/gomod" go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
```

Run lint:

```bash
./.bin/golangci-lint run
```

Install GoReleaser locally:

```bash
GOBIN="$PWD/.bin" GOCACHE="$PWD/.cache/go-build-host" GOMODCACHE="$PWD/.cache/gomod-host" go install github.com/goreleaser/goreleaser/v2@latest
```

Run the full minimum verification suite:

```bash
go vet ./...
./.bin/golangci-lint run
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
- Match the local lint toolchain to CI when investigating CI failures; today CI
  resolved `golangci-lint` to `v1.64.8`
- When changing the release pipeline, run a local GoReleaser snapshot plus both
  MCPB asset build commands before declaring success
- Add or update focused tests when behavior changes
- Keep docs aligned with the actual command surface in `cmd/`
