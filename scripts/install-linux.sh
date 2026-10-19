#!/usr/bin/env bash
set -euo pipefail

CORE_URL="${1:-}"
VERSION="${2:-0.1.0}"
REPO="${3:-Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent}"
INSTALL_DIR="${4:-/opt/emir-agent}"
UPDATE_AT_STARTUP="${UPDATE_AT_STARTUP:-true}"
UPDATE_BOOT_DELAY_SECONDS="${UPDATE_BOOT_DELAY_SECONDS:-90}"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ASSET_NAME="emir-agent-linux-amd64.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/$ASSET_NAME"
SERVICE_NAME="emir-agent"
STATE_DIR="/var/lib/emir-agent"
STATE_PATH="$STATE_DIR/state.json"

if [ "$EUID" -ne 0 ]; then
    echo "This script must be run as root."
    exit 1
fi

echo "Stopping existing service..."
systemctl stop "$SERVICE_NAME" 2>/dev/null || true
systemctl stop emir-agent-update-boot.timer 2>/dev/null || true
systemctl disable emir-agent-update-boot.timer 2>/dev/null || true
rm -f /etc/systemd/system/emir-agent-update-boot.service /etc/systemd/system/emir-agent-update-boot.timer

echo "Downloading $ASSET_NAME..."
TMP_DIR=$(mktemp -d)
curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ASSET_NAME"

echo "Extracting to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$INSTALL_DIR"
mv -f "$INSTALL_DIR/emir-agent-linux-amd64" "$INSTALL_DIR/emir-agent"
mv -f "$INSTALL_DIR/emir-agent-popup-linux-amd64" "$INSTALL_DIR/emir-agent-popup"
chmod +x "$INSTALL_DIR/emir-agent"
chmod +x "$INSTALL_DIR/emir-agent-popup"

# Copy boot update helper and record install metadata for recovery.
cp "$SCRIPT_DIR/update-at-boot.sh" "$INSTALL_DIR/update-at-boot.sh"
chmod +x "$INSTALL_DIR/update-at-boot.sh"
cat > "$INSTALL_DIR/install.conf" <<EOF
REPO=$REPO
VERSION=$VERSION
ARCHIVE=emir-agent-linux-amd64.tar.gz
EOF

if [ -n "$CORE_URL" ]; then
    echo "EMIR_CORE_URL=$CORE_URL" > /etc/default/emir-agent
fi

# Use a machine-wide state directory so the service can access the same state
# and credentials created during interactive pairing.
mkdir -p "$STATE_DIR"
chmod 700 "$STATE_DIR"
echo "EMIR_STATE_PATH=$STATE_PATH" >> /etc/default/emir-agent
export EMIR_STATE_PATH="$STATE_PATH"

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

if [ "$UPDATE_AT_STARTUP" = "true" ]; then
    cat > /etc/systemd/system/emir-agent-update-boot.service <<EOF
[Unit]
Description=EMIR Agent update check at boot
After=network.target

[Service]
Type=oneshot
ExecStart=$INSTALL_DIR/update-at-boot.sh
EnvironmentFile=-/etc/default/emir-agent
EOF

    cat > /etc/systemd/system/emir-agent-update-boot.timer <<EOF
[Unit]
Description=Run EMIR Agent update check at boot
Requires=emir-agent-update-boot.service

[Timer]
OnBootSec=${UPDATE_BOOT_DELAY_SECONDS}s
Unit=emir-agent-update-boot.service

[Install]
WantedBy=timers.target
EOF

    systemctl daemon-reload
    systemctl enable emir-agent-update-boot.timer
    systemctl start emir-agent-update-boot.timer || true
    echo "Registered emir-agent-update-boot.timer (delay: ${UPDATE_BOOT_DELAY_SECONDS}s)."
fi

echo "emir-agent installed and running."
rm -rf "$TMP_DIR"
