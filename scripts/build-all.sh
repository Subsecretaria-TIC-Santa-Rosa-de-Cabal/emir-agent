#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0}"
OUTPUT_DIR="${2:-dist}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

bash "$SCRIPT_DIR/build-linux.sh" "$VERSION" "$OUTPUT_DIR"
bash "$SCRIPT_DIR/build-macos.sh" "$VERSION" "$OUTPUT_DIR"
powershell -ExecutionPolicy Bypass -File "$SCRIPT_DIR/build-windows.ps1" -Version "$VERSION" -OutputDir "$OUTPUT_DIR"
