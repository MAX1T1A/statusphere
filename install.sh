#!/usr/bin/env bash
set -euo pipefail

REPO="https://github.com/MAX1T1A/statusphere.git"
BINARY="statusphere"
INSTALL_DIR="$HOME/.local/bin"
TMP_DIR="$(mktemp -d)"

cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

if ! command -v go &>/dev/null; then
    echo "error: go is not installed"
    echo "install it: https://go.dev/dl/"
    exit 1
fi

if ! command -v git &>/dev/null; then
    echo "error: git is not installed"
    exit 1
fi

echo "cloning statusphere..."
git clone --depth 1 "$REPO" "$TMP_DIR/statusphere"

echo "building..."
cd "$TMP_DIR/statusphere/client"
go build -o "$BINARY" ./cmd/client

echo "installing to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
mv "$BINARY" "$INSTALL_DIR/$BINARY"

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