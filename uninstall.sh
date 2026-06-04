#!/bin/sh
# gp-cli macOS/Linux uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/noaa/patent-cli/main/uninstall.sh | sh

set -e

BINARY="gp-cli"
INSTALL_DIR="$HOME/.local/bin"
CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/Library/Application Support}/patent-cli"

# macOS config dir
if [ "$(uname -s)" = "Darwin" ]; then
  CONFIG_DIR="$HOME/Library/Application Support/patent-cli"
else
  CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/patent-cli"
fi

BINARY_PATH="$INSTALL_DIR/$BINARY"
REMOVED=0

if [ -f "$BINARY_PATH" ]; then
  rm -f "$BINARY_PATH"
  echo ">> Removed: $BINARY_PATH"
  REMOVED=1
else
  echo ">> Binary not found: $BINARY_PATH"
fi

if [ -d "$CONFIG_DIR" ]; then
  rm -rf "$CONFIG_DIR"
  echo ">> Removed config: $CONFIG_DIR"
  REMOVED=1
else
  echo ">> Config dir not found: $CONFIG_DIR"
fi

if [ "$REMOVED" = "1" ]; then
  echo ""
  echo ">> gp-cli uninstalled."
  echo ">> You may also remove the PATH entry from your ~/.zshrc or ~/.bashrc:"
  echo "   export PATH=\"\$PATH:$INSTALL_DIR\""
else
  echo ""
  echo ">> Nothing to remove."
fi
