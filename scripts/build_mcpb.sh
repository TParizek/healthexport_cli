#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d)"

VERSION="${HE_MCPB_VERSION:-0.0.0-dev}"
MANIFEST_VERSION="${VERSION#v}"
GOOS="${GOOS:-darwin}"
GOARCH="${GOARCH:-$(go env GOARCH)}"
OUT_FILE="${1:-$ROOT_DIR/dist/health-export_${MANIFEST_VERSION}_${GOOS}_${GOARCH}.mcpb}"
if [[ "$OUT_FILE" != /* ]]; then
  OUT_FILE="$ROOT_DIR/$OUT_FILE"
fi

cleanup() {
  rm -rf "$TMP_DIR"
}

trap cleanup EXIT

mkdir -p "$TMP_DIR/server" "$(dirname "$OUT_FILE")"

pushd "$ROOT_DIR" >/dev/null
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
  -ldflags "-s -w -X main.version=$MANIFEST_VERSION -X main.commit=${HE_MCPB_COMMIT:-none} -X main.date=${HE_MCPB_DATE:-unknown}" \
  -o "$TMP_DIR/server/he-mcp" \
  ./cmd/he-mcp
sed "s/__VERSION__/$MANIFEST_VERSION/g" packaging/mcp/manifest.json.tmpl > "$TMP_DIR/manifest.json"
if [[ -f packaging/mcp/icon.png ]]; then
  cp packaging/mcp/icon.png "$TMP_DIR/icon.png"
fi
if [[ -d packaging/mcp/skills ]]; then
  cp -R packaging/mcp/skills "$TMP_DIR/skills"
fi
popd >/dev/null

(
  cd "$TMP_DIR"
  ARCHIVE_INPUTS=(manifest.json server)
  if [[ -f icon.png ]]; then
    ARCHIVE_INPUTS+=(icon.png)
  fi
  if [[ -d skills ]]; then
    ARCHIVE_INPUTS+=(skills)
  fi
  zip -qr "$OUT_FILE" "${ARCHIVE_INPUTS[@]}"
)

echo "Wrote $OUT_FILE"
