#!/usr/bin/env bash
set -e

# Local development test for emir-agent auto-update on Linux/macOS.
# Builds an "old" and a "new" agent binary, starts a local HTTP server to
# serve the new binary, and prints the SQL needed to register the fake
# release in emir-core.

CORE_URL="${1:-http://localhost:8000}"
OLD_VERSION="${2:-0.3.0}"
NEW_VERSION="${3:-0.3.5}"

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$REPO_ROOT/dist-test"
SERVER_PORT=9999

mkdir -p "$DIST_DIR"

echo "Building old agent version $OLD_VERSION..."
OLD_BINARY="$DIST_DIR/emir-agent-old"
GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) go build \
    -ldflags "-s -w -X github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models.Version=$OLD_VERSION" \
    -o "$OLD_BINARY" "$REPO_ROOT"

echo "Building new agent version $NEW_VERSION..."
NEW_BINARY="$DIST_DIR/emir-agent-new"
GOOS=$(go env GOOS) GOARCH=$(go env GOARCH) go build \
    -ldflags "-s -w -X github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent/internal/models.Version=$NEW_VERSION" \
    -o "$NEW_BINARY" "$REPO_ROOT"

CHECKSUM=$(sha256sum "$NEW_BINARY" | awk '{print $1}')
echo "New binary checksum: $CHECKSUM"

echo "Starting local HTTP server on port $SERVER_PORT..."
cd "$DIST_DIR"
python3 -m http.server "$SERVER_PORT" > /tmp/emir-agent-update-server.log 2>&1 &
SERVER_PID=$!
sleep 2

DOWNLOAD_URL="http://localhost:$SERVER_PORT/emir-agent-new"

echo ""
echo "=== SQL to register the fake release in emir-core ==="
cat <<EOF
INSERT INTO agent_releases (
    id, version, download_url, checksum, is_mandatory, release_notes, enabled, active, registration_date, last_update
) VALUES (
    gen_random_uuid(),
    '$NEW_VERSION',
    '$DOWNLOAD_URL',
    '$CHECKSUM',
    true,
    'Local test update',
    true,
    true,
    now(),
    now()
);
EOF
echo ""

echo "=== Next steps ==="
echo "1. Run the SQL above in your emir-core database."
echo "2. Ensure emir-core is running at $CORE_URL."
echo "3. Pair the old agent if needed:"
echo "   $OLD_BINARY --pair"
echo "4. Run the old agent and watch for the update:"
echo "   EMIR_CORE_URL=$CORE_URL $OLD_BINARY"
echo "5. After the update, verify the running binary version:"
echo "   $NEW_BINARY --version  # if supported, or check logs"
echo ""

read -p "Press Enter to stop the local HTTP server"
kill "$SERVER_PID" 2>/dev/null || true
