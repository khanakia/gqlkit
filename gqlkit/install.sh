#!/bin/sh
set -e

REPO="khanakia/gqlkit"
BINARY="gqlkit"
INSTALL_DIR="/usr/local/bin"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS" && exit 1 ;;
esac

URL="https://github.com/${REPO}/releases/latest/download/${BINARY}_${OS}_${ARCH}.tar.gz"

echo "Downloading ${BINARY} for ${OS}/${ARCH}..."
tmpdir=$(mktemp -d)
curl -sL "$URL" | tar xz -C "$tmpdir"

echo "Installing to ${INSTALL_DIR}/${BINARY}..."
sudo mv "$tmpdir/$BINARY" "$INSTALL_DIR/$BINARY"
rm -rf "$tmpdir"

echo "Done! Run 'gqlkit version' to verify."
