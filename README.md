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
/plugin marketplace add betidestu/neostack-plugins
/plugin install neostack-connect@neostack
/reload-plugins
```

Verify with `/mcp` — you should see `neostack` connected.

**macOS / Linux only**: run the one-time setup so the plugin selects the right native proxy binary for your OS:

```sh
~/.claude/plugins/cache/neostack/neostack-connect/*/setup.sh
```

Then run `/reload-plugins` or restart Claude Code. Windows works out of the box.

### Codex CLI

```bash
codex plugin marketplace add betidestu/neostack-plugins
codex
```

Inside Codex, open `/plugins`, find **neostack-connect**, press Space to enable.

Run the one-time setup from the installed plugin directory. Codex prints the installed plugin root after `codex plugin add neostack-connect@neostack`; run the setup script from that exact folder. Codex needs this on every OS because its plugin loader does not substitute plugin-root variables in MCP command paths, and NeoStack discovery needs Codex to keep the working directory on your Unreal project.

Windows:

```bat
cd %USERPROFILE%\.codex\plugins\cache\neostack\neostack-connect\0.1.4
setup.cmd
```

macOS / Linux:

```sh
cd ~/.codex/plugins/cache/neostack/neostack-connect/0.1.4
./setup.sh
```

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
    ├── .mcp.json                       # active Claude Code MCP config
    ├── claude*.mcp.json                # one per OS (setup copies the active config)
    ├── codex*.mcp.json                 # platform templates
    ├── setup.sh / setup.cmd            # generates active Claude/Codex configs per OS
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

Claude Desktop release bundles are built as `.mcpb` files with `.dxt` aliases so both the current MCPB tooling and Claude Desktop's in-app Extension installer are covered.

## Source repo

Edits land here, not in the editor's source repo. The editor (private) and the marketplace (public) are decoupled — schema changes coordinate via the `runtime.json` `schemaVersion` field, currently `2`.
