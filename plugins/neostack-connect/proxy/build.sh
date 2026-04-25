#!/bin/sh
# Cross-compile neostack-mcp-proxy for all supported platforms.
# Output: ../bin/<platform>/neostack-mcp-proxy[.exe]
#
# Usage:
#   ./build.sh            # all targets
#   ./build.sh win64      # only one
set -e

cd "$(dirname "$0")"
BIN_ROOT="$(cd .. && pwd)/bin"

LDFLAGS="-s -w"

build_one() {
  local platform=$1 goos=$2 goarch=$3 ext=$4
  if [ -n "$ONLY" ] && [ "$ONLY" != "$platform" ]; then return 0; fi
  mkdir -p "$BIN_ROOT/$platform"
  out="$BIN_ROOT/$platform/neostack-mcp-proxy$ext"
  echo "Building $goos/$goarch -> $out"
  GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build -trimpath -ldflags="$LDFLAGS" -o "$out" .
}

ONLY="${1:-}"
build_one win64        windows amd64 .exe
build_one macos-arm64  darwin  arm64 ""
build_one macos-x64    darwin  amd64 ""
build_one linux-x64    linux   amd64 ""

echo "Done."
