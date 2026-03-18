#!/usr/bin/env bash
set -euo pipefail

REPO="MAX1T1A/statusphere"
BINARY="statusphere"
INSTALL_DIR="$HOME/.local/bin"

ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    *)
        echo "error: unsupported architecture: $ARCH"
        exit 1
        ;;
esac

ASSET="${BINARY}-linux-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

echo "downloading ${ASSET}..."
if command -v curl &>/dev/null; then
    curl -fsSL -o "/tmp/${BINARY}" "$URL"
elif command -v wget &>/dev/null; then
    wget -qO "/tmp/${BINARY}" "$URL"
else
    echo "error: curl or wget required"
    exit 1
fi

chmod +x "/tmp/${BINARY}"
mkdir -p "$INSTALL_DIR"
mv "/tmp/${BINARY}" "$INSTALL_DIR/$BINARY"

SHELL_RC=""
case "$(basename "$SHELL")" in
    zsh)  SHELL_RC="$HOME/.zshrc" ;;
    bash) SHELL_RC="$HOME/.bashrc" ;;
    fish) SHELL_RC="$HOME/.config/fish/config.fish" ;;
esac

if [ -n "$SHELL_RC" ] && [ -f "$SHELL_RC" ]; then
    if ! grep -q "alias ss=" "$SHELL_RC" 2>/dev/null; then
        echo "" >> "$SHELL_RC"
        echo "# statusphere" >> "$SHELL_RC"
        if [ "$(basename "$SHELL")" = "fish" ]; then
            echo "alias ss '$INSTALL_DIR/$BINARY'" >> "$SHELL_RC"
        else
            echo "alias ss='$INSTALL_DIR/$BINARY'" >> "$SHELL_RC"
        fi
        echo "added alias ss → $SHELL_RC"
    fi

    if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
        if [ "$(basename "$SHELL")" = "fish" ]; then
            echo "set -gx PATH $INSTALL_DIR \$PATH" >> "$SHELL_RC"
        else
            echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$SHELL_RC"
        fi
    fi
fi

echo ""
echo "done! restart your shell or run:"
echo "  source $SHELL_RC"
echo ""
echo "then:"
echo "  ss --register https://your-server.com"