#!/bin/sh
# Configures codex.mcp.json for the current OS+arch.
# Run once after installing the plugin in Codex on Mac/Linux.
# Windows users can use setup.cmd or skip — the shipped codex.mcp.json defaults to Windows.

set -e
DIR="$(cd "$(dirname "$0")" && pwd)"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64)  SRC=codex-macos-arm64.mcp.json ;;
  Darwin-x86_64) SRC=codex-macos-x64.mcp.json   ;;
  Linux-x86_64)  SRC=codex-linux-x64.mcp.json   ;;
  *) echo "Unsupported platform: $(uname -s) $(uname -m)" >&2; exit 1 ;;
esac

if [ ! -f "$DIR/$SRC" ]; then
  echo "Missing $SRC — was the plugin built with all targets?" >&2
  exit 1
fi

cp "$DIR/$SRC" "$DIR/codex.mcp.json"

# Make the binary executable (zip extraction loses +x on some platforms).
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64)  chmod +x "$DIR/bin/macos-arm64/neostack-mcp-proxy"  ;;
  Darwin-x86_64) chmod +x "$DIR/bin/macos-x64/neostack-mcp-proxy"    ;;
  Linux-x86_64)  chmod +x "$DIR/bin/linux-x64/neostack-mcp-proxy"    ;;
esac

# Clear macOS quarantine so Gatekeeper doesn't block it.
case "$(uname -s)" in
  Darwin) xattr -dr com.apple.quarantine "$DIR/bin" 2>/dev/null || true ;;
esac

echo "Configured codex.mcp.json from $SRC. Run \`codex /reload-plugins\` (or restart Codex)."
