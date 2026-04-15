#!/bin/sh
# Install script for Hector (macOS / Linux)
# Usage:
#   curl -fsSL https://gohector.dev/install.sh | sh
#   curl -fsSL https://gohector.dev/install.sh | sh -s -- --version v1.2.3
#   curl -fsSL https://gohector.dev/install.sh | sh -s -- --install-dir /usr/local/bin
#
# For Windows (PowerShell), use scripts/install.ps1 instead:
#   irm https://gohector.dev/install.ps1 | iex

set -e

GITHUB_REPO="verikod/hector"
BINARY_NAME="hector"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION=""

# Parse arguments
while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)   VERSION="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux)   OS="linux" ;;
  Darwin)  OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS" >&2
    echo "On Windows, use: irm https://gohector.dev/install.ps1 | iex" >&2
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Detect available download tool
if command -v curl > /dev/null 2>&1; then
  DOWNLOAD="curl -fsSL"
elif command -v wget > /dev/null 2>&1; then
  DOWNLOAD="wget -qO-"
else
  echo "Error: curl or wget is required." >&2
  exit 1
fi

# Resolve latest version if not specified
if [ -z "$VERSION" ]; then
  echo "Fetching latest release..."
  VERSION="$(
    $DOWNLOAD "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/'
  )"
  if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version. Check your network or specify --version." >&2
    exit 1
  fi
fi

# Strip leading 'v' for the archive filename (goreleaser uses the version without 'v')
VERSION_NUMBER="${VERSION#v}"

# Build archive name
ARCHIVE="${BINARY_NAME}_${VERSION_NUMBER}_${OS}_${ARCH}.tar.gz"

DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "Installing ${BINARY_NAME} ${VERSION} (${OS}/${ARCH})..."
echo "  from: ${DOWNLOAD_URL}"

# Download and extract to a temp directory
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

$DOWNLOAD "$DOWNLOAD_URL" | tar -xz -C "$TMP_DIR"

# Install binary
BINARY_PATH="${TMP_DIR}/${BINARY_NAME}"
if [ ! -f "$BINARY_PATH" ]; then
  echo "Error: binary not found in archive." >&2
  exit 1
fi

chmod +x "$BINARY_PATH"

# Create install dir if needed (no-op if it already exists)
if [ ! -d "$INSTALL_DIR" ]; then
  mkdir -p "$INSTALL_DIR" 2>/dev/null || {
    echo "Error: cannot create ${INSTALL_DIR}. Try running with sudo or set --install-dir." >&2
    exit 1
  }
fi

# Move binary into place, using sudo only if needed
if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY_PATH" "${INSTALL_DIR}/${BINARY_NAME}"
else
  echo "  (requires sudo to write to ${INSTALL_DIR})"
  sudo mv "$BINARY_PATH" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo ""
echo "Installed: ${INSTALL_DIR}/${BINARY_NAME}"
"${INSTALL_DIR}/${BINARY_NAME}" version 2>/dev/null || true
echo ""
echo "Run 'hector --help' to get started."
