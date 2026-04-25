package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const heartbeatStaleWindow = 30 * time.Second

type runtimeFile struct {
	SchemaVersion   int             `json:"schemaVersion"`
	InstanceID      string          `json:"instanceId"`
	EditorPID       int             `json:"editorPid"`
	ProjectName     string          `json:"projectName"`
	ProjectPath     string          `json:"projectPath"`
	UprojectPath    string          `json:"uprojectPath"`
	PluginVersion   string          `json:"pluginVersion"`
	EngineVersion   string          `json:"engineVersion"`
	StartedAt       string          `json:"startedAt"`
	LastHeartbeatAt string          `json:"lastHeartbeatAt"`
	MCPRunning      bool            `json:"mcpRunning"`
	MCPServers      []runtimeServer `json:"mcpServers"`
	IDEConnected    bool            `json:"ideConnected"`
}

type runtimeServer struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

type discoveredEditor struct {
	URL          string
	ProjectName  string
	InstanceID   string
	UprojectPath string
}

// discoveryError carries a user-facing message and an optional actionable hint.
type discoveryError struct {
	msg  string
	hint string
}

func (e *discoveryError) Error() string { return e.msg }

func newDiscoveryError(msg, hint string) error {
	return &discoveryError{msg: msg, hint: hint}
}

func findProjectDir() (string, error) {
	if env := os.Getenv("NEOSTACK_PROJECT_DIR"); env != "" {
		if _, err := os.Stat(env); err != nil {
			return "", newDiscoveryError(
				fmt.Sprintf("NEOSTACK_PROJECT_DIR points at a path that does not exist: %s", env),
				"",
			)
		}
		abs, err := filepath.Abs(env)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				if strings.EqualFold(filepath.Ext(e.Name()), ".uproject") {
					return dir, nil
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", newDiscoveryError(
		"Could not find an Unreal project (.uproject) by walking up from the current directory.",
		"Run claude/codex from inside your UE project directory, or set NEOSTACK_PROJECT_DIR to its absolute path.",
	)
}

func discover() (*discoveredEditor, error) {
	projectDir, err := findProjectDir()
	if err != nil {
		return nil, err
	}

	runtimePath := filepath.Join(projectDir, "Saved", "NeoStackAI", "runtime.json")
	data, err := os.ReadFile(runtimePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, newDiscoveryError(
				fmt.Sprintf("No NeoStackAI runtime file at %s.", runtimePath),
				"Open the Unreal editor for this project with the NeoStackAI plugin enabled.",
			)
		}
		return nil, err
	}

	var rt runtimeFile
	if err := json.Unmarshal(data, &rt); err != nil {
		return nil, newDiscoveryError(
			fmt.Sprintf("runtime.json is not valid JSON: %v", err),
			"",
		)
	}

	if rt.SchemaVersion != 2 {
		return nil, newDiscoveryError(
			fmt.Sprintf("runtime.json schemaVersion is %d, expected 2.", rt.SchemaVersion),
			"Update the NeoStackAI plugin in your editor to a version that writes schemaVersion 2.",
		)
	}

	if !rt.MCPRunning {
		return nil, newDiscoveryError(
			fmt.Sprintf("NeoStackAI MCP server is not running for project '%s'.", rt.ProjectName),
			"Make sure the editor is fully loaded and the MCP server is enabled in NeoStackAI settings.",
		)
	}

	heartbeat, err := time.Parse(time.RFC3339Nano, rt.LastHeartbeatAt)
	if err != nil {
		return nil, newDiscoveryError(
			fmt.Sprintf("runtime.json lastHeartbeatAt is unparseable: %q", rt.LastHeartbeatAt),
			"",
		)
	}
	age := time.Since(heartbeat)
	if age > heartbeatStaleWindow {
		return nil, newDiscoveryError(
			fmt.Sprintf("Editor heartbeat is stale (%ds old).", int(age.Seconds())),
			"The editor may have crashed or be unresponsive. Restart it.",
		)
	}

	for _, s := range rt.MCPServers {
		if s.Type == "http" {
			return &discoveredEditor{
				URL:          s.URL,
				ProjectName:  rt.ProjectName,
				InstanceID:   rt.InstanceID,
				UprojectPath: rt.UprojectPath,
			}, nil
		}
	}

	return nil, newDiscoveryError(
		"runtime.json has no MCP server with type='http'.",
		"This proxy currently only supports HTTP transport.",
	)
}
