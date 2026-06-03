#!/bin/sh
# Build per-platform Desktop Extensions for Claude Desktop.
# Produces .mcpb files and .dxt aliases in dist/, one per OS+arch.
#
# Prerequisites:
#   npm install -g @anthropic-ai/mcpb
#   bash ../proxy/build.sh   # build the Go binaries first

set -e
cd "$(dirname "$0")"

VERSION="${VERSION:-0.1.6}"
PLUGIN_ROOT="$(cd .. && pwd)"
BIN_ROOT="$PLUGIN_ROOT/bin"
DIST="$PWD/dist"

rm -rf "$DIST" stage
mkdir -p "$DIST"

# (platform-tag, GOOS for platform_overrides, source-binary, dest-binary-name)
build_one() {
  local plat="$1" os="$2" src_bin="$3" dest_bin="$4"
  local stage="$PWD/stage/$plat"

  if [ ! -f "$BIN_ROOT/$plat/$src_bin" ]; then
    echo "::warning::Missing binary $BIN_ROOT/$plat/$src_bin — run proxy/build.sh first" >&2
    return 0
  fi

  rm -rf "$stage"
  mkdir -p "$stage/server"
  cp "$BIN_ROOT/$plat/$src_bin" "$stage/server/$dest_bin"

  # Make Unix binaries executable inside the bundle.
  case "$os" in
    darwin|linux) chmod +x "$stage/server/$dest_bin" ;;
  esac

  # Per-platform manifest. Uses ${__dirname} so the binary path resolves correctly
  # regardless of where Claude Desktop installs the extracted bundle.
  cat > "$stage/manifest.json" <<MANIFEST
{
  "manifest_version": "0.3",
  "name": "neostack-connect",
  "display_name": "NeoStack Connect",
  "version": "$VERSION",
  "description": "Connect Claude Desktop to a running NeoStackAI Unreal editor.",
  "long_description": "Bridges Claude Desktop's MCP support to the HTTP MCP server inside an open Unreal Engine session that has the NeoStackAI plugin enabled. Set a project folder to pin this extension to one project, or leave it unset to auto-connect when exactly one NeoStackAI-enabled editor is running.",
  "author": {
    "name": "NeoStack",
    "url": "https://neostack.dev"
  },
  "homepage": "https://neostack.dev",
  "documentation": "https://neostack.dev/docs/claude-desktop",
  "support": "https://discord.gg/betide",
  "repository": {
    "type": "git",
    "url": "https://github.com/betidestu/neostack-plugins"
  },
  "license": "MIT",
  "keywords": ["unreal", "ue5", "neostack", "mcp"],
  "tools": [
    {
      "name": "execute_script",
      "description": "Run a NeoStack Lua script in the connected Unreal editor."
    },
    {
      "name": "unreal_status",
      "description": "Report why NeoStack Connect cannot reach the Unreal editor when discovery fails."
    },
    {
      "name": "list_unreal_projects",
      "description": "List active NeoStackAI-enabled Unreal editor projects when discovery is ambiguous."
    }
  ],
  "tools_generated": true,
  "server": {
    "type": "binary",
    "entry_point": "server/$dest_bin",
    "mcp_config": {
      "command": "\${__dirname}/server/$dest_bin",
      "args": [],
      "env": {
        "NEOSTACK_PROJECT_DIR": "\${user_config.project_dir}"
      }
    }
  },
  "user_config": {
    "project_dir": {
      "type": "directory",
      "title": "Unreal project directory",
      "description": "Optional path to the folder that contains your .uproject file. Set this when multiple Unreal editors may be open.",
      "required": false
    }
  },
  "compatibility": {
    "claude_desktop": ">=0.10.0",
    "platforms": ["$os"]
  }
}
MANIFEST

  mcpb validate "$stage/manifest.json"

  # Stable filenames (no embedded version) so docs can link to
  # /releases/latest/download/neostack-connect-<plat>.mcpb forever.
  local out="$DIST/neostack-connect-$plat.mcpb"
  mcpb pack "$stage" "$out"
  mcpb clean "$out"
  cp "$out" "$DIST/neostack-connect-$plat.dxt"
  echo "Packed $out"
}

build_one win64       win32  neostack-mcp-proxy.exe neostack-mcp-proxy.exe
build_one macos-arm64 darwin neostack-mcp-proxy     neostack-mcp-proxy
build_one macos-x64   darwin neostack-mcp-proxy     neostack-mcp-proxy
build_one linux-x64   linux  neostack-mcp-proxy     neostack-mcp-proxy

rm -rf stage
echo ""
echo "Done. Artifacts:"
ls -lh "$DIST"
