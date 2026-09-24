#!/usr/bin/env bash
set -euo pipefail

CORE_URL="${1:-}"
VERSION="${2:-0.1.0}"
REPO="${3:-alcaldia/emir-agent}"
INSTALL_DIR="${4:-/usr/local/bin}"

ASSET_NAME="emir-agent-macos-amd64.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/$ASSET_NAME"
SERVICE_NAME="com.emir.agent"

if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root."
    exit 1
fi

echo "Stopping existing service..."
launchctl unload /Library/LaunchDaemons/$SERVICE_NAME.plist 2>/dev/null || true

echo "Downloading $ASSET_NAME..."
TMP_DIR=$(mktemp -d)
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET_NAME"

echo "Extracting to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$INSTALL_DIR"
mv -f "$INSTALL_DIR/emir-agent-macos-amd64" "$INSTALL_DIR/emir-agent"
chmod +x "$INSTALL_DIR/emir-agent"

if [ -n "$CORE_URL" ]; then
    launchctl setenv EMIR_CORE_URL "$CORE_URL"
fi

echo "Pairing the agent with emir-core..."
"$INSTALL_DIR/emir-agent" --pair

if [ $? -ne 0 ]; then
    echo "Pairing failed. The service will not be started."
    exit 1
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

echo "emir-agent installed and running."
rm -rf "$TMP_DIR"
