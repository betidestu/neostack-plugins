#!/bin/sh
# Build per-platform .mcpb Desktop Extensions for Claude Desktop.
# Produces 4 .mcpb files in dist/, one per OS+arch.
#
# Prerequisites:
#   npm install -g @anthropic-ai/mcpb
#   bash ../proxy/build.sh   # build the Go binaries first

set -e
cd "$(dirname "$0")"

VERSION="${VERSION:-0.1.0}"
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
  "long_description": "Bridges Claude Desktop's MCP support to the HTTP MCP server inside an open Unreal Engine session that has the NeoStackAI plugin enabled. You point it at your project folder once at install; from then on it auto-discovers the editor whenever it's running.",
  "author": {
    "name": "NeoStack",
    "url": "https://neostack.dev"
  },
  "homepage": "https://neostack.dev",
  "repository": {
    "type": "git",
    "url": "https://github.com/betidestudio/neostack-plugins"
  },
  "license": "MIT",
  "keywords": ["unreal", "ue5", "neostack", "mcp"],
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
      "description": "Path to the folder that contains your .uproject file. The proxy reads <project>/Saved/NeoStackAI/runtime.json to find the running editor.",
      "required": true
    }
  },
  "compatibility": {
    "platforms": ["$os"]
  }
}
MANIFEST

  local out="$DIST/neostack-connect-$plat-$VERSION.mcpb"
  mcpb pack "$stage" "$out"
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
