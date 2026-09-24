#!/usr/bin/env bash
set -euo pipefail

CORE_URL="${1:-}"
VERSION="${2:-0.1.0}"
REPO="${3:-alcaldia/emir-agent}"
INSTALL_DIR="${4:-/opt/emir-agent}"

ASSET_NAME="emir-agent-linux-amd64.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/$ASSET_NAME"
SERVICE_NAME="emir-agent"

if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root."
    exit 1
fi

echo "Stopping existing service..."
systemctl stop "$SERVICE_NAME" 2>/dev/null || true

echo "Downloading $ASSET_NAME..."
TMP_DIR=$(mktemp -d)
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET_NAME"

echo "Extracting to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$INSTALL_DIR"
mv -f "$INSTALL_DIR/emir-agent-linux-amd64" "$INSTALL_DIR/emir-agent"
chmod +x "$INSTALL_DIR/emir-agent"

if [ -n "$CORE_URL" ]; then
    echo "EMIR_CORE_URL=$CORE_URL" > /etc/default/emir-agent
fi

echo "Pairing the agent with emir-core..."
"$INSTALL_DIR/emir-agent" --pair

if [ $? -ne 0 ]; then
    echo "Pairing failed. The service will not be started."
    exit 1
fi

cat > /etc/systemd/system/$SERVICE_NAME.service <<EOF
[Unit]
Description=EMIR Agent
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/emir-agent
Restart=always
RestartSec=60
EnvironmentFile=-/etc/default/emir-agent

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"

echo "emir-agent installed and running."
rm -rf "$TMP_DIR"
