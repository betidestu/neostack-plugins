package main

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"
)

// runDiagnosticMode is the proxy's fallback when discovery fails. Instead of
// exiting (which leaves the LLM with a generic "MCP failed" error and no
// recourse), it serves a minimal stdio MCP server that exposes a single tool
// returning the discovery error. The LLM can call that tool and relay the
// fix to the user in plain language.
func runDiagnosticMode(de *discoveryError) {
	body := de.msg
	if de.hint != "" {
		body = "neostack-connect is not connected to an Unreal editor.\n\n" +
			"Reason: " + de.msg + "\n\n" +
			"Fix: " + de.hint
	}

	var writeMu sync.Mutex
	enc := json.NewEncoder(os.Stdout)

	send := func(payload any) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = enc.Encode(payload)
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)

	for scanner.Scan() {
		var req map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		method, _ := req["method"].(string)
		id, hasID := req["id"]

		switch method {
		case "initialize":
			send(map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"protocolVersion": "2025-11-25",
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo": map[string]any{
						"name":    "neostack-connect (diagnostic)",
						"version": "0.1.7",
					},
				},
			})
		case "notifications/initialized":
			// notification — no response

		case "tools/list":
			send(map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name": "unreal_status",
							"description": "Reports why neostack-connect can't reach the Unreal editor. " +
								"Call this whenever a user asks about Unreal Engine, NeoStackAI, or expects editor tools — the actual editor tools are not available right now.",
							"inputSchema": map[string]any{
								"type":       "object",
								"properties": map[string]any{},
							},
						},
						{
							"name":        "list_unreal_projects",
							"description": "Lists active NeoStackAI-enabled Unreal editor projects discovered on this machine.",
							"inputSchema": map[string]any{
								"type":       "object",
								"properties": map[string]any{},
							},
						},
					},
				},
			})

		case "tools/call":
			name := ""
			if params, ok := req["params"].(map[string]any); ok {
				name, _ = params["name"].(string)
			}

			text := body
			isError := true
			if name == "list_unreal_projects" {
				text = discoveredProjectsText()
				isError = false
			}

			send(map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"result": map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": text},
					},
					"isError": isError,
				},
			})

		default:
			if hasID && id != nil {
				send(map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"error": map[string]any{
						"code":    -32601,
						"message": "Method not available in diagnostic mode",
					},
				})
			}
		}
	}
}

func discoveredProjectsText() string {
	editors, err := activeEditorsFromRegistry()
	if err != nil {
		return err.Error()
	}
	return formatEditorChoices(editors)
}
