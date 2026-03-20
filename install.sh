#!/bin/sh
set -e

REPO="dosaki/claude-sessions"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Darwin) OS="darwin" ;;
    Linux)  OS="linux" ;;
    *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)             echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release tag
TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')"
if [ -z "$TAG" ]; then
    echo "Error: could not determine latest release"
    exit 1
fi

echo "Installing claude-sessions ${TAG} (${OS}/${ARCH})..."

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

if [ "$OS" = "darwin" ]; then
    ASSET="claude-sessions-darwin-${ARCH}.zip"
    curl -fsSL "https://github.com/${REPO}/releases/download/${TAG}/${ASSET}" -o "${TMP}/${ASSET}"
    unzip -q "${TMP}/${ASSET}" -d "${TMP}"

    echo "Installing Claude Sessions.app to /Applications..."
    if [ -d "/Applications/Claude Sessions.app" ]; then
        rm -rf "/Applications/Claude Sessions.app"
    fi
    cp -R "${TMP}/Claude Sessions.app" /Applications/

    echo "Symlinking binary to ${INSTALL_DIR}/claude-sessions..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo ln -sf "/Applications/Claude Sessions.app/Contents/MacOS/claude-sessions" "${INSTALL_DIR}/claude-sessions"
else
    # Linux
    ASSET="claude-sessions-linux-${ARCH}"
    curl -fsSL "https://github.com/${REPO}/releases/download/${TAG}/${ASSET}" -o "${TMP}/claude-sessions"
    chmod +x "${TMP}/claude-sessions"

    echo "Installing to ${INSTALL_DIR}/claude-sessions..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo mv "${TMP}/claude-sessions" "${INSTALL_DIR}/claude-sessions"
fi

echo "Done! Run 'claude-sessions' to get started."
