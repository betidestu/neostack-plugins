# NeoStack Plugins

Marketplace of plugins for [Claude Code](https://docs.claude.com/en/docs/claude-code) and [Codex](https://developers.openai.com/codex) that connect to a running [NeoStackAI](https://neostack.dev) Unreal Engine session.

## Install

### Claude Code

From a terminal inside your UE project directory:

```bash
claude
```

Then inside Claude Code:

```
/plugin marketplace add betidestudio/neostack-plugins
/plugin install neostack-connect@neostack
/reload-plugins
```

Verify with `/mcp` — you should see `neostack` connected. No platform-specific setup needed.

### Codex CLI

```bash
codex plugin marketplace add betidestudio/neostack-plugins
codex
```

Inside Codex, open `/plugins`, find **neostack-connect**, press Space to enable.

**macOS / Linux only**: run the one-time setup so Codex picks the right binary for your OS:

```sh
~/.codex/plugins/neostack-connect/setup.sh
```

(Windows works out of the box.)

## What's here

- `plugins/neostack-connect/` — the only plugin. Spawns a Go-compiled stdio MCP proxy (~6 MB per platform) that auto-discovers your running editor by walking up from cwd to find `<project>/Saved/NeoStackAI/runtime.json`, then bridges every tool call to the editor's HTTP MCP server.

## Layout

```
.
├── .claude-plugin/marketplace.json     # Claude Code reads this
├── .agents/plugins/marketplace.json    # Codex reads this
└── plugins/neostack-connect/
    ├── .claude-plugin/plugin.json
    ├── .codex-plugin/plugin.json
    ├── .mcp.json                       # Claude Code uses ${CLAUDE_PLUGIN_ROOT}
    ├── codex*.mcp.json                 # one per OS (Codex doesn't substitute env vars)
    ├── setup.sh / setup.cmd            # picks the right Codex config per OS
    ├── bin/<platform>/neostack-mcp-proxy[.exe]
    └── proxy/                          # Go source — `bash build.sh` to recompile
```

## Building from source

Requires Go 1.23+:

```bash
cd plugins/neostack-connect/proxy
bash build.sh             # all platforms
bash build.sh win64       # one platform
```

## Source repo

Edits land here, not in the editor's source repo. The editor (private) and the marketplace (public) are decoupled — schema changes coordinate via the `runtime.json` `schemaVersion` field, currently `2`.
