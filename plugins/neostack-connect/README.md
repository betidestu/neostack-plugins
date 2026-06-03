# neostack-connect

Stdio MCP proxy that bridges Claude Code / Codex to a running NeoStackAI Unreal editor.

## How discovery works

When the editor is open with NeoStackAI loaded, it writes `<ProjectDir>/Saved/NeoStackAI/runtime.json` with the live MCP server URL and a heartbeat timestamp. It also publishes active editor sessions to a machine-level registry:

- Windows: `%LOCALAPPDATA%\NeoStackAI\runtimes.json`
- macOS / Linux: `~/.neostack/runtimes.json`

The proxy:

1. Walks up from the current working directory looking for a `.uproject` file. Override with `NEOSTACK_PROJECT_DIR=<abs-path>`.
2. Reads `<ProjectDir>/Saved/NeoStackAI/runtime.json` when a project context is available.
3. Falls back to the machine-level runtime registry when no project context exists, or when the registry has a fresh entry for the current project.
4. Auto-connects only when discovery resolves to exactly one active editor. If several editors are active and no project context disambiguates them, diagnostic mode reports the available project paths.
5. Validates the heartbeat is fresh (< 30s old), `mcpRunning` is true, and connects to the first `http`-type MCP server.
6. Bridges stdio MCP frames to that HTTP endpoint for the lifetime of the session.

## Failure modes

The proxy exits with a clear message if:

- No `.uproject` is found by walking up from cwd
- The runtime file doesn't exist (editor not running or NeoStackAI not loaded)
- Multiple active editors are running and no project context was available
- The heartbeat is stale (editor crashed or unresponsive)
- The MCP server isn't running inside the editor

## Platform setup

Claude Code expands `${CLAUDE_PLUGIN_ROOT}` and `${CLAUDE_PROJECT_DIR}`, but it does not select platform-specific binary paths for us. Codex's plugin loader also does not substitute `${...}` in plugin MCP configs and only resolves `cwd` against plugin root.

So we ship pre-built configs for each supported platform:

- `claude-windows.mcp.json`, `claude-macos-arm64.mcp.json`, `claude-macos-x64.mcp.json`, `claude-linux-x64.mcp.json`
- `codex-windows.mcp.json`, `codex-macos-arm64.mcp.json`, `codex-macos-x64.mcp.json`, `codex-linux-x64.mcp.json`

The setup script copies the right Claude config into `.mcp.json`. For Codex, it generates `codex.mcp.json` with an absolute proxy binary path and no `cwd`; that keeps Codex's process working directory on the active Unreal project so `.uproject` discovery works.

**Windows**: from the plugin's install dir, run once:

```bat
setup.cmd
```

**Mac / Linux**: from the plugin's install dir, run once:

```sh
./setup.sh
```

This swaps in the right configs, marks the binary executable, and clears macOS Gatekeeper quarantine. Then run `/reload-plugins` in Claude Code/Codex, or restart the app.

Codex prints the installed plugin root after `codex plugin add neostack-connect@neostack`. For marketplace installs, it is usually:

- Windows: `%USERPROFILE%\.codex\plugins\cache\neostack\neostack-connect\0.1.6`
- macOS / Linux: `~/.codex/plugins/cache/neostack/neostack-connect/0.1.6`

Tracking upstream — env var expansion in Codex MCP config: [openai/codex#2680](https://github.com/openai/codex/issues/2680). Once that lands we collapse the four configs back into one.

## Building from source

```bash
cd proxy
bash build.sh              # cross-compiles to ../bin/{win64,macos-arm64,macos-x64,linux-x64}/
bash build.sh win64        # only one target
```
