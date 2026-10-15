#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0}"
OUTPUT_DIR="${2:-dist}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/$OUTPUT_DIR"

mkdir -p "$OUTPUT_DIR"

export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
export GOTOOLCHAIN=local

BINARY_NAME="emir-agent-linux-amd64"
BINARY_PATH="$OUTPUT_DIR/$BINARY_NAME"

echo "Building $BINARY_NAME (version $VERSION)..."
go build -ldflags "-s -w -X github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models.Version=$VERSION" -o "$BINARY_PATH" "$PROJECT_ROOT"

cp "$SCRIPT_DIR/update-at-boot.sh" "$OUTPUT_DIR/update-at-boot.sh"

TAR_PATH="$OUTPUT_DIR/$BINARY_NAME.tar.gz"
tar -czf "$TAR_PATH" -C "$OUTPUT_DIR" "$BINARY_NAME" "update-at-boot.sh"

echo "Built: $BINARY_PATH"
echo "Packaged: $TAR_PATH"

sha256sum "$BINARY_PATH" | awk '{print $1}' > "$BINARY_PATH.sha256"
echo "Checksum: $(cat "$BINARY_PATH.sha256")"
