#!/usr/bin/env bash
set -euo pipefail

# EMIR Agent update check at boot (Linux).
# If the agent binary is present it invokes emir-agent --update-check to apply
# mandatory updates. If the binary is missing it enters recovery mode and
# re-downloads the version recorded at install time.

INSTALL_DIR="${INSTALL_DIR:-/opt/emir-agent}"
LOG_DIR="${LOG_DIR:-/var/log/emir-agent}"
STATE_DIR="${STATE_DIR:-/var/lib/emir-agent}"

BINARY="$INSTALL_DIR/emir-agent"
LOG_FILE="$LOG_DIR/update-boot.log"
LOCK_FILE="$STATE_DIR/update.lock"
META_FILE="$INSTALL_DIR/install.conf"
LOCK_TIMEOUT_MINUTES=5

if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root." >&2
    exit 1
fi

mkdir -p "$LOG_DIR" "$STATE_DIR"

log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> "$LOG_FILE"
}

read_meta() {
    REPO="Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent"
    VERSION=""
    ASSET="emir-agent-linux-amd64"

    if [ -f "$META_FILE" ]; then
        # shellcheck source=/dev/null
        source "$META_FILE"
    fi
}

acquire_lock() {
    if [ -f "$LOCK_FILE" ]; then
        local age_min
        age_min=$(( ( $(date +%s) - $(stat -c %Y "$LOCK_FILE") ) / 60 ))
        if [ "$age_min" -lt "$LOCK_TIMEOUT_MINUTES" ]; then
            log "Another update is in progress (lock age: ${age_min} min). Exiting."
            exit 0
        fi
    fi
    touch "$LOCK_FILE"
}

release_lock() {
    rm -f "$LOCK_FILE"
}

recover_agent() {
    read_meta
    if [ -z "$VERSION" ]; then
        log "Cannot recover agent: no version recorded in $META_FILE"
        return 1
    fi

    local tag asset url checksum_url
    tag="v$VERSION"
    asset="$ASSET"
    url="https://github.com/$REPO/releases/download/$tag/$asset"
    checksum_url="$url.sha256"

    log "Recovery mode: downloading $VERSION from $url"

    local tmpdir
    tmpdir=$(mktemp -d)
    # shellcheck disable=SC2064
    trap "rm -rf '$tmpdir'" RETURN

    curl -fsSL -A "emir-agent-boot-updater" -o "$tmpdir/$asset" "$url"

    if curl -fsSL -A "emir-agent-boot-updater" -o "$tmpdir/$asset.sha256" "$checksum_url" 2>/dev/null; then
        local expected actual
        expected=$(tr -d '[:space:]' < "$tmpdir/$asset.sha256")
        actual=$(sha256sum "$tmpdir/$asset" | awk '{print $1}')
        if [ "$actual" != "$expected" ]; then
            log "Recovery checksum mismatch: expected $expected, got $actual"
            return 1
        fi
        log "Recovery checksum verified"
    else
        log "Could not download recovery checksum; proceeding without verification"
    fi

    mkdir -p "$INSTALL_DIR"
    mv -f "$tmpdir/$asset" "$BINARY"
    chmod +x "$BINARY"
    log "Agent binary restored to $BINARY"
}

run_update_check() {
    log "Running $BINARY --update-check"
    if "$BINARY" --update-check; then
        log "emir-agent --update-check finished successfully"
    else
        local code=$?
        log "emir-agent --update-check failed with exit code $code"
    fi
}

log "Starting EMIR Agent boot update check"
acquire_lock
trap release_lock EXIT

if [ ! -f "$BINARY" ]; then
    log "Agent binary not found at $BINARY; entering recovery mode"
    recover_agent || exit 1
else
    run_update_check
fi
