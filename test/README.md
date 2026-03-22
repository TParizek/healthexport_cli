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
