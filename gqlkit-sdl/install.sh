#!/bin/sh
set -e

REPO="khanakia/gqlkit"
BINARY="gqlkit-sdl"
TAG_PREFIX="gqlkit-sdl@"
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

# Find the latest release matching this tool's tag prefix
TAG=$(curl -sL "https://api.github.com/repos/${REPO}/releases" \
  | grep -o "\"tag_name\": *\"${TAG_PREFIX}v[^\"]*\"" \
  | head -1 \
  | cut -d'"' -f4)

if [ -z "$TAG" ]; then
  echo "Error: No release found for ${BINARY}" && exit 1
fi

ENCODED_TAG="$TAG"
ASSET="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${ENCODED_TAG}/${ASSET}"

echo "Downloading ${BINARY} (${TAG})..."
tmpdir=$(mktemp -d)
curl -sL "$URL" | tar xz -C "$tmpdir"

echo "Installing to ${INSTALL_DIR}/${BINARY}..."
sudo mv "$tmpdir/$BINARY" "$INSTALL_DIR/$BINARY"
rm -rf "$tmpdir"

echo "Done! Run 'gqlkit-sdl version' to verify."
