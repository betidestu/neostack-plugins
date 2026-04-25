# neostack-connect

Stdio MCP proxy that bridges Claude Code / Codex to a running NeoStackAI Unreal editor.

## How discovery works

When the editor is open with NeoStackAI loaded, it writes `<ProjectDir>/Saved/NeoStackAI/runtime.json` with the live MCP server URL and a heartbeat timestamp. The proxy:

1. Walks up from the current working directory looking for a `.uproject` file. Override with `NEOSTACK_PROJECT_DIR=<abs-path>`.
2. Reads `<ProjectDir>/Saved/NeoStackAI/runtime.json`.
3. Validates the heartbeat is fresh (< 30s old) and `mcpRunning` is true.
4. Connects to the first `http`-type MCP server in the file.
5. Bridges stdio MCP frames to that HTTP endpoint for the lifetime of the session.

## Failure modes

The proxy exits with a clear message if:

- No `.uproject` is found by walking up from cwd
- The runtime file doesn't exist (editor not running or NeoStackAI not loaded)
- The heartbeat is stale (editor crashed or unresponsive)
- The MCP server isn't running inside the editor

## Codex setup (one-time, Mac/Linux only)

Codex's plugin loader (per `codex-rs/core-plugins/src/loader.rs`) doesn't substitute `${...}` in `.mcp.json` and only resolves the `cwd` field against plugin root — there's no way to write a single Codex config that works on every OS. So we ship four pre-built configs (`codex-windows.mcp.json`, `codex-macos-arm64.mcp.json`, `codex-macos-x64.mcp.json`, `codex-linux-x64.mcp.json`) and a `setup` script that copies the right one into `codex.mcp.json`.

**Windows**: nothing to do — the shipped `codex.mcp.json` already targets Windows. (You can run `setup.cmd` to make it explicit.)

**Mac / Linux**: from the plugin's install dir, run once:

```sh
./setup.sh
```

This swaps in the right config, marks the binary executable, and clears macOS Gatekeeper quarantine. Then `codex /reload-plugins` (or restart Codex) and you're set.

Tracking upstream — env var expansion in Codex MCP config: [openai/codex#2680](https://github.com/openai/codex/issues/2680). Once that lands we collapse the four configs back into one.

Claude Code has none of this — it expands `${CLAUDE_PLUGIN_ROOT}` natively, so `.mcp.json` works cross-platform with no setup step.

## Building from source

```bash
cd proxy
bun install
bun run build              # cross-compiles to bin/{win64,macos-arm64,macos-x64,linux-x64}/
bun run build win64        # only one target
bun run dev                # run from source against current cwd
```
