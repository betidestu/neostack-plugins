#!/bin/sh
# Configures Claude Code and Codex MCP configs for the current OS+arch.
# Run once after installing the plugin on Mac/Linux. Windows users should use
# setup.cmd so Codex gets an absolute proxy command path.
# Codex gets a generated absolute proxy path so the process cwd remains the
# active Unreal project directory for .uproject discovery.

set -e
DIR="$(cd "$(dirname "$0")" && pwd)"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64)  CLAUDE_SRC=claude-macos-arm64.mcp.json; CODEX_BIN=bin/macos-arm64/neostack-mcp-proxy ;;
  Darwin-x86_64) CLAUDE_SRC=claude-macos-x64.mcp.json;   CODEX_BIN=bin/macos-x64/neostack-mcp-proxy   ;;
  Linux-x86_64)  CLAUDE_SRC=claude-linux-x64.mcp.json;   CODEX_BIN=bin/linux-x64/neostack-mcp-proxy   ;;
  *) echo "Unsupported platform: $(uname -s) $(uname -m)" >&2; exit 1 ;;
esac

if [ ! -f "$DIR/$CLAUDE_SRC" ] || [ ! -f "$DIR/$CODEX_BIN" ]; then
  echo "Missing platform MCP config — was the plugin built with all targets?" >&2
  exit 1
fi

cp "$DIR/$CLAUDE_SRC" "$DIR/.mcp.json"
CODEX_PROXY="$DIR/$CODEX_BIN"
JSON_PROXY=$(printf '%s' "$CODEX_PROXY" | sed 's/\\/\\\\/g; s/"/\\"/g')
cat > "$DIR/codex.mcp.json" <<EOF
{
  "mcpServers": {
    "neostack": {
      "command": "$JSON_PROXY"
    }
  }
}
EOF

if command -v codex >/dev/null 2>&1; then
  codex mcp add neostack -- "$CODEX_PROXY" >/dev/null
  echo "Registered neostack with Codex using an absolute proxy path."
else
  echo "Codex CLI not found on PATH; generated codex.mcp.json only."
fi

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

echo "Configured .mcp.json from $CLAUDE_SRC and codex.mcp.json for $CODEX_PROXY."
echo "Run \`/reload-plugins\` in Claude Code/Codex, or restart the app."
