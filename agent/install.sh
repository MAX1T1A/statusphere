#!/usr/bin/env bash
set -euo pipefail

# Installs statusphere as a headless agent on a server: system user, binary,
# join the room, mark the machine as a server, configs, systemd unit.
# See ./README.md for what each step does and why.
#
# Usage (as root):
#   ./install.sh <invite-code> ["Display Name"]
# or piped:
#   curl -sSL https://raw.githubusercontent.com/MAX1T1A/statusphere/master/agent/install.sh \
#     | sudo bash -s -- <invite-code> "Display Name"
#
# Safe to re-run: skips steps that are already done instead of clobbering them.

REPO="MAX1T1A/statusphere"
BINARY="statusphere"
SERVICE_USER="statusphere"
SERVICE_HOME="/home/statusphere"
RAW="https://raw.githubusercontent.com/${REPO}/master/agent"
BIN_PATH="/usr/local/bin/${BINARY}"

if [ "$(id -u)" -ne 0 ]; then
    echo "error: run as root (sudo)" >&2
    exit 1
fi

INVITE="${1:-}"
NAME="${2:-}"
if [ -z "$INVITE" ]; then
    echo "usage: $0 <invite-code> [\"display name\"]" >&2
    exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *)
        echo "error: unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

echo "==> system user"
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd --system --create-home --home-dir "$SERVICE_HOME" "$SERVICE_USER"
else
    echo "    $SERVICE_USER already exists"
fi

echo "==> binary"
ASSET="${BINARY}-linux-${ARCH}"
curl -fsSL -o "/tmp/${BINARY}" "https://github.com/${REPO}/releases/latest/download/${ASSET}"
chmod +x "/tmp/${BINARY}"
install -m755 "/tmp/${BINARY}" "$BIN_PATH"
rm -f "/tmp/${BINARY}"

CONFIG_DIR="${SERVICE_HOME}/.config/statusphere"
CONFIG_FILE="${CONFIG_DIR}/config.json"

echo "==> account"
if [ -f "$CONFIG_FILE" ]; then
    echo "    already joined a room, skipping --join (remove $CONFIG_FILE to rejoin)"
else
    sudo -u "$SERVICE_USER" -H "$BIN_PATH" --join "$INVITE"
fi

if [ -n "$NAME" ]; then
    sudo -u "$SERVICE_USER" -H "$BIN_PATH" --set-name "$NAME"
fi
sudo -u "$SERVICE_USER" -H "$BIN_PATH" --set-kind server

echo "==> config"
mkdir -p "$CONFIG_DIR"
for f in privacy custom; do
    if [ ! -f "${CONFIG_DIR}/${f}.json" ]; then
        curl -fsSL -o "${CONFIG_DIR}/${f}.json" "${RAW}/${f}.example.json"
    else
        echo "    ${f}.json already present, leaving it alone"
    fi
done
chown -R "${SERVICE_USER}:${SERVICE_USER}" "$CONFIG_DIR"
chmod 600 "${CONFIG_DIR}"/*.json

echo "==> systemd unit"
UNIT="/etc/systemd/system/statusphere-agent.service"
if [ ! -f "$UNIT" ]; then
    curl -fsSL -o "$UNIT" "${RAW}/statusphere-agent.service"
else
    echo "    unit already installed, leaving it alone"
fi
systemctl daemon-reload
systemctl enable --now statusphere-agent

echo ""
echo "done. custom.json in ${CONFIG_DIR} is a template - edit it for this box's checks."
echo "watch it: journalctl -u statusphere-agent -f"
