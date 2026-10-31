#!/bin/sh
# Ventox One-Line Installer for Linux & macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/0xzahed/ventox/main/scripts/install.sh | sh

set -e

REPO="0xzahed/ventox"
TAG="${VENTOX_VERSION:-latest}"

# Color codes
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

printf "${CYAN}⚡ Ventox Universal Installer${NC}\n"
printf "Detecting host operating system and CPU architecture...\n"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux*)  TARGET_OS="linux" ;;
  darwin*) TARGET_OS="darwin" ;;
  *)
    printf "${RED}Error: Unsupported operating system: $OS${NC}\n"
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64) TARGET_ARCH="amd64" ;;
  arm64|aarch64) TARGET_ARCH="arm64" ;;
  *)
    printf "${RED}Error: Unsupported CPU architecture: $ARCH${NC}\n"
    exit 1
    ;;
esac

TARBALL="ventox-${TARGET_OS}-${TARGET_ARCH}.tar.gz"

if [ "$TAG" = "latest" ]; then
  DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${TARBALL}"
else
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${TARBALL}"
fi

printf "Identified platform: ${GREEN}${TARGET_OS}/${TARGET_ARCH}${NC}\n"
printf "Downloading: ${CYAN}${DOWNLOAD_URL}${NC}\n"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$TARBALL"
tar -xzf "$TMP_DIR/$TARBALL" -C "$TMP_DIR"

# Determine install directory
INSTALL_DIR="/usr/local/bin"
USE_SUDO=""

if [ ! -w "$INSTALL_DIR" ]; then
  if command -v sudo >/dev/null 2>&1; then
    USE_SUDO="sudo"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi
fi

printf "Installing ventox binary into ${GREEN}${INSTALL_DIR}${NC}...\n"
$USE_SUDO mv "$TMP_DIR/ventox-${TARGET_OS}-${TARGET_ARCH}" "$INSTALL_DIR/ventox" 2>/dev/null || \
$USE_SUDO mv "$TMP_DIR/ventox" "$INSTALL_DIR/ventox"
$USE_SUDO chmod +x "$INSTALL_DIR/ventox"

printf "\n${GREEN}✔ Successfully installed Ventox!${NC}\n"
printf "Run ${CYAN}ventox --help${NC} or ${CYAN}ventox list${NC} to get started.\n"
