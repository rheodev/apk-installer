#!/usr/bin/env sh
set -eu

BIN_PATH="${1:-}"
if [ -z "$BIN_PATH" ]; then
  echo "copy-platform-tools: missing Wails binary path"
  exit 1
fi

case "$BIN_PATH" in
  *.app/Contents/MacOS/*)
    OS="darwin"
    ;;
  *.exe)
    OS="windows"
    ;;
  *)
    OS="$(go env GOOS)"
    ;;
esac
SRC="../../third_party/platform-tools/$OS"

if [ ! -d "$SRC" ]; then
  echo "copy-platform-tools: skip, $SRC not found"
  exit 0
fi

case "$OS" in
  darwin)
    DEST="$(dirname "$BIN_PATH")/../Resources/platform-tools"
    ;;
  windows)
    DEST="$(dirname "$BIN_PATH")/platform-tools"
    ;;
  *)
    DEST="$(dirname "$BIN_PATH")/platform-tools"
    ;;
esac

rm -rf "$DEST"
mkdir -p "$DEST"
cp -R "$SRC/." "$DEST/"

if [ "$OS" = "darwin" ] || [ "$OS" = "linux" ]; then
  if [ -f "$DEST/adb" ]; then
    chmod +x "$DEST/adb"
  fi
fi

echo "copy-platform-tools: copied $SRC to $DEST"
