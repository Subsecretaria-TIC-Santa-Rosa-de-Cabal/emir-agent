#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0}"
OUTPUT_DIR="${2:-dist}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/$OUTPUT_DIR"

mkdir -p "$OUTPUT_DIR"

export GOOS=darwin
export GOARCH=amd64
export GOTOOLCHAIN=local

AGENT_BINARY_NAME="emir-agent-macos-amd64"
AGENT_BINARY_PATH="$OUTPUT_DIR/$AGENT_BINARY_NAME"
POPUP_BINARY_NAME="emir-agent-popup-macos-amd64"
POPUP_BINARY_PATH="$OUTPUT_DIR/$POPUP_BINARY_NAME"

AGENT_LDFLAGS="-s -w -X github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models.Version=$VERSION"
POPUP_LDFLAGS="-s -w -X main.Version=$VERSION"

echo "Building $AGENT_BINARY_NAME (version $VERSION)..."
export CGO_ENABLED=0
go build -ldflags "$AGENT_LDFLAGS" -o "$AGENT_BINARY_PATH" "$PROJECT_ROOT"

echo "Building $POPUP_BINARY_NAME (version $VERSION)..."
CGO_ENABLED=1 go build -ldflags "$POPUP_LDFLAGS" -o "$POPUP_BINARY_PATH" "$PROJECT_ROOT/cmd/popup"

cp "$SCRIPT_DIR/update-at-boot-macos.sh" "$OUTPUT_DIR/update-at-boot-macos.sh"

TAR_PATH="$OUTPUT_DIR/$AGENT_BINARY_NAME.tar.gz"
tar -czf "$TAR_PATH" -C "$OUTPUT_DIR" "$AGENT_BINARY_NAME" "$POPUP_BINARY_NAME" "update-at-boot-macos.sh"

echo "Built: $AGENT_BINARY_PATH"
echo "Built: $POPUP_BINARY_PATH"
echo "Packaged: $TAR_PATH"

shasum -a 256 "$AGENT_BINARY_PATH" | awk '{print $1}' > "$AGENT_BINARY_PATH.sha256"
echo "Agent checksum: $(cat "$AGENT_BINARY_PATH.sha256")"

shasum -a 256 "$POPUP_BINARY_PATH" | awk '{print $1}' > "$POPUP_BINARY_PATH.sha256"
echo "Popup checksum: $(cat "$POPUP_BINARY_PATH.sha256")"
