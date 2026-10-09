# Integration Testing

## Crypto Test Vectors

`testdata/test_vectors.json` is generated from the backend crypto
implementation and committed to this repository as a compatibility fixture.

To regenerate it:

1. Open `requirements/tasks/16-integration-testing.md`.
2. Run the `generate_test_vectors.js` snippet in the backend repository root.
3. Save the JSON output to `testdata/test_vectors.json`.
4. Run `go test ./...` to confirm the auth and crypto tests still match the
   backend behavior.

The current vectors were derived from the backend implementations in:

- `models/crypto/HERDecoder.js`
- `models/crypto/Chacha20.js`

## Manual End-to-End Script

Build the CLI, then run the manual real-API script with a valid account key:

```bash
go build -o he
HEALTHEXPORT_TEST_ACCOUNT_KEY=your-key bash test/e2e_test.sh
```

The script uses a temporary `XDG_CONFIG_HOME` so it does not overwrite your
real CLI config.

## Cross-Platform Checklist

The integration task is intentionally manual. Minimum verification for each
target platform is:

- Build or obtain the platform binary.
- Run `he version`.
- Run `he types`.

Suggested targets:

- macOS arm64
- macOS amd64
- Linux amd64
- Windows amd64

## Workout details

`go test ./...` exercises a synthetic encrypted workout backend through service,
CLI command, and local MCP protocol tests. It covers complete pagination,
revision conflicts, authenticated envelope identity, malformed or missing pages,
unit conversion, unavailable measurements, and output formats. Late-page failures
must leave stdout empty.

Compatibility fixtures under `internal/service/testdata/`:

- `workout-detail-ios.json`: synthetic AES-GCM envelopes emitted by the iOS
  Swift CryptoKit exporter, copied from HE-MCP's interoperability fixture.
- `workout-web-export.json`: generated using HE-FE-Remote's
  `src/models/WorkoutDetailExport.ts` `createWorkoutDetailJsonExport` function
  with the synthetic data in `internal/workouttest/fixture.go`. Regenerate with
  that website helper if the export contract changes, then run
  `TestWorkoutExportMatchesWebsiteGolden`. Do not derive the expected fixture
  from the CLI exporter itself.

For a read-only live check, use an existing CLI login:

```bash
./he workouts list --from 2026-10-01 --to 2026-10-10 --format json
# Replace the placeholders with an ID and revision from the listing.
./he workouts export <workout-id> --revision <revision> > workout.json
./he workouts series <workout-id> --revision <revision> --section heartRate --format csv > heart-rate.csv
```

Choose a range containing your uploaded workouts. Compare measurement counts,
splits, and values with the website export. Keep account keys and real health
records out of committed fixtures. The automated suite uses only synthetic data.
