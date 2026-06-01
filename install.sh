#!/bin/sh
# gp-cli macOS/Linux installer
# Usage: curl -fsSL https://raw.githubusercontent.com/noaa/patent-cli/main/install.sh | sh
# Options:
#   --source   Build and install from source via 'go install' (requires Go 1.21+)

set -e

REPO="noaa/patent-cli"
BINARY="gp-cli"
INSTALL_DIR="$HOME/.local/bin"
BUILD_FROM_SOURCE=0

for arg in "$@"; do
  case "$arg" in
    --source) BUILD_FROM_SOURCE=1 ;;
    *) echo "Unknown option: $arg"; exit 1 ;;
  esac
done

check_path() {
  case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
      echo ">> NOTE: Add the following line to your ~/.zshrc or ~/.bashrc:"
      echo ""
      echo "   export PATH=\"\$PATH:$INSTALL_DIR\""
      echo ""
      echo "   Then run: source ~/.zshrc"
      ;;
  esac
}

# ── Build from source ────────────────────────────────────────────────────────
if [ "$BUILD_FROM_SOURCE" = "1" ]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "Error: 'go' not found. Install Go 1.21+ from https://go.dev/dl/ and retry."
    exit 1
  fi
  echo ">> Building from source (go install)..."
  mkdir -p "$INSTALL_DIR"
  GOBIN="$INSTALL_DIR" go install "github.com/${REPO}/cmd/${BINARY}@latest"
  echo ""
  echo ">> Installed: $INSTALL_DIR/$BINARY"
  echo ""
  check_path
  echo ">> Done! Run: $BINARY --help"
  exit 0
fi

# ── Pre-built binary from GitHub Releases ────────────────────────────────────

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  darwin)  PLATFORM="darwin" ;;
  linux)   PLATFORM="linux" ;;
  *)       echo "Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
  x86_64)          GOARCH="amd64" ;;
  arm64|aarch64)   GOARCH="arm64" ;;
  *)               echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

ASSET="${BINARY}-${PLATFORM}-${GOARCH}"

# Fetch latest release tag
echo ">> Fetching latest release..."
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')

if [ -z "$TAG" ]; then
  echo "Error: Could not find latest release. Check https://github.com/${REPO}/releases"
  exit 1
fi

echo ">> Installing ${BINARY} ${TAG} for ${PLATFORM}/${GOARCH}"

URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}.tar.gz"

# Download and extract
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "$TMP/${ASSET}.tar.gz"
tar xzf "$TMP/${ASSET}.tar.gz" -C "$TMP"

# Install
mkdir -p "$INSTALL_DIR"
install -m 755 "$TMP/${ASSET}" "$INSTALL_DIR/${BINARY}"

echo ""
echo ">> Installed: $INSTALL_DIR/$BINARY"
echo ""
check_path
echo ">> Done! Run: $BINARY --help"
