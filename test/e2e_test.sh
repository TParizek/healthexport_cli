#!/bin/bash
set -euo pipefail

HE="${HE:-./he}"
ACCOUNT_KEY="${HEALTHEXPORT_TEST_ACCOUNT_KEY:?Set HEALTHEXPORT_TEST_ACCOUNT_KEY}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

expect_exit_code() {
  local expected="$1"
  shift

  set +e
  "$@"
  local status=$?
  set -e

  if [[ "$status" -ne "$expected" ]]; then
    echo "Expected exit code $expected, got $status: $*" >&2
    exit 1
  fi
}

require_cmd jq
require_cmd awk

if [[ ! -x "$HE" ]]; then
  echo "Binary not found or not executable: $HE" >&2
  echo "Build it first with: go build -o he" >&2
  exit 1
fi

CONFIG_HOME="$(mktemp -d "${TMPDIR:-/tmp}/healthexport-e2e.XXXXXX")"
trap 'rm -rf "$CONFIG_HOME"' EXIT
export XDG_CONFIG_HOME="$CONFIG_HOME"

echo "=== Auth flow ==="
printf '%s\n' "$ACCOUNT_KEY" | "$HE" auth login
"$HE" auth status
"$HE" auth logout

echo "=== Types ==="
"$HE" types | awk 'NR <= 5 { print }'
"$HE" types --format json | jq '.[0]'
"$HE" types --category aggregated | awk 'NR <= 5 { print }'

echo "=== Data (CSV) ==="
"$HE" data --type step_count --from 2024-01-01 --to 2024-01-07 --account-key "$ACCOUNT_KEY"

echo "=== Data (JSON) ==="
"$HE" data --type step_count --from 2024-01-01 --to 2024-01-07 --format json --account-key "$ACCOUNT_KEY" | jq .

echo "=== Data (raw) ==="
"$HE" data --type 9 --from 2024-01-01 --to 2024-01-07 --raw --account-key "$ACCOUNT_KEY" | jq .

echo "=== Data (aggregated) ==="
"$HE" data --type step_count --from 2024-01-01 --to 2024-01-31 --aggregate day --account-key "$ACCOUNT_KEY"

echo "=== Data (multiple types) ==="
"$HE" data --type step_count --type body_mass --from 2024-01-01 --to 2024-01-07 --account-key "$ACCOUNT_KEY"

echo "=== Environment variable auth ==="
HEALTHEXPORT_ACCOUNT_KEY="$ACCOUNT_KEY" "$HE" data --type 9 --from 2024-01-01 --to 2024-01-07

echo "=== Error cases ==="
expect_exit_code 4 "$HE" data --type nonexistent --from 2024-01-01 --to 2024-01-07 --account-key "$ACCOUNT_KEY"
expect_exit_code 4 "$HE" data --type body_mass --from 2024-01-01 --to 2024-01-07 --aggregate day --account-key "$ACCOUNT_KEY"

echo "=== Agent patterns ==="
"$HE" data --type step_count --from 2024-01-01 --to 2024-01-07 --format json --account-key "$ACCOUNT_KEY" 2>/dev/null | jq length
"$HE" data --type step_count --from 2024-01-01 --to 2024-01-07 --account-key "$ACCOUNT_KEY" 2>/dev/null | awk -F, 'NR > 1 { sum += $5 } END { print "Total:", sum }'

TYPE_ID=$("$HE" types --format json | jq -r '.[] | select(.name == "Step count") | .id' | head -n 1)
if [[ -z "$TYPE_ID" ]]; then
  echo "Failed to resolve type ID for Step count" >&2
  exit 1
fi

"$HE" data --type "$TYPE_ID" --from 2024-01-01 --to 2024-01-07 --account-key "$ACCOUNT_KEY" >/dev/null

set +e
"$HE" data --type nonexistent --from 2024-01-01 --to 2024-01-07 --account-key "$ACCOUNT_KEY" >/dev/null 2>&1
STATUS=$?
set -e

echo "Exit: $STATUS"
if [[ "$STATUS" -ne 4 ]]; then
  echo "Expected exit code 4 for unknown type, got $STATUS" >&2
  exit 1
fi

echo "=== All tests passed ==="
