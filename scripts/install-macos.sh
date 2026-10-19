#!/usr/bin/env bash
set -euo pipefail

CORE_URL="${1:-}"
VERSION="${2:-0.1.0}"
REPO="${3:-Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent}"
INSTALL_DIR="${4:-/usr/local/bin}"
UPDATE_AT_STARTUP="${UPDATE_AT_STARTUP:-true}"
UPDATE_BOOT_DELAY_SECONDS="${UPDATE_BOOT_DELAY_SECONDS:-90}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ASSET_NAME="emir-agent-macos-amd64.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/$ASSET_NAME"
SERVICE_NAME="com.emir.agent"
STATE_DIR="/Library/Application Support/emir-agent"
STATE_PATH="$STATE_DIR/state.json"

if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root."
    exit 1
fi

echo "Stopping existing service..."
launchctl unload /Library/LaunchDaemons/$SERVICE_NAME.plist 2>/dev/null || true
launchctl unload /Library/LaunchDaemons/${SERVICE_NAME}.update.plist 2>/dev/null || true
rm -f /Library/LaunchDaemons/${SERVICE_NAME}.update.plist

echo "Downloading $ASSET_NAME..."
TMP_DIR=$(mktemp -d)
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET_NAME"

echo "Extracting to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$INSTALL_DIR"
mv -f "$INSTALL_DIR/emir-agent-macos-amd64" "$INSTALL_DIR/emir-agent"
mv -f "$INSTALL_DIR/emir-agent-popup-macos-amd64" "$INSTALL_DIR/emir-agent-popup"
chmod +x "$INSTALL_DIR/emir-agent"
chmod +x "$INSTALL_DIR/emir-agent-popup"

# Copy boot update helper and record install metadata for recovery.
mkdir -p /usr/local/etc/emir-agent
cp "$SCRIPT_DIR/update-at-boot-macos.sh" "$INSTALL_DIR/update-at-boot-macos.sh"
chmod +x "$INSTALL_DIR/update-at-boot-macos.sh"
cat > /usr/local/etc/emir-agent/install.conf <<EOF
REPO=$REPO
VERSION=$VERSION
ARCHIVE=emir-agent-macos-amd64.tar.gz
EOF

if [ -n "$CORE_URL" ]; then
    launchctl setenv EMIR_CORE_URL "$CORE_URL"
fi

# Use a machine-wide state directory so the launchd job can access the same
# state and credentials created during interactive pairing.
mkdir -p "$STATE_DIR"
chmod 700 "$STATE_DIR"
launchctl setenv EMIR_STATE_PATH "$STATE_PATH"
export EMIR_STATE_PATH="$STATE_PATH"

echo "Pairing the agent with emir-core..."
"$INSTALL_DIR/emir-agent" --pair

if [ $? -ne 0 ]; then
    echo "Pairing failed. The service will not be started."
    exit 1
fi

CORE_ENV_PLIST=""
if [ -n "$CORE_URL" ]; then
    CORE_ENV_PLIST="<key>EMIR_CORE_URL</key><string>$CORE_URL</string>"
fi

cat > /Library/LaunchDaemons/$SERVICE_NAME.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$SERVICE_NAME</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/emir-agent</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>EMIR_STATE_PATH</key>
        <string>$STATE_PATH</string>
        $CORE_ENV_PLIST
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/emir-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/emir-agent.error.log</string>
</dict>
</plist>
EOF

launchctl unload /Library/LaunchDaemons/$SERVICE_NAME.plist 2>/dev/null || true
launchctl load /Library/LaunchDaemons/$SERVICE_NAME.plist

if [ "$UPDATE_AT_STARTUP" = "true" ]; then
    cat > /Library/LaunchDaemons/${SERVICE_NAME}.update.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${SERVICE_NAME}.update</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/update-at-boot-macos.sh</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>EMIR_STATE_PATH</key>
        <string>$STATE_PATH</string>
        <key>UPDATE_BOOT_DELAY_SECONDS</key>
        <string>$UPDATE_BOOT_DELAY_SECONDS</string>
        $CORE_ENV_PLIST
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>LaunchOnlyOnce</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/emir-agent-update-boot.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/emir-agent-update-boot.error.log</string>
</dict>
</plist>
EOF

    launchctl unload /Library/LaunchDaemons/${SERVICE_NAME}.update.plist 2>/dev/null || true
    launchctl load /Library/LaunchDaemons/${SERVICE_NAME}.update.plist
    echo "Registered ${SERVICE_NAME}.update LaunchDaemon (delay: ${UPDATE_BOOT_DELAY_SECONDS}s)."
fi

echo "emir-agent installed and running."
rm -rf "$TMP_DIR"
