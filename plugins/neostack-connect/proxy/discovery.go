package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

type registryFile struct {
	SchemaVersion int           `json:"schemaVersion"`
	UpdatedAt     string        `json:"updatedAt"`
	Runtimes      []runtimeFile `json:"runtimes"`
}

type discoveredEditor struct {
	URL             string
	ProjectName     string
	ProjectPath     string
	InstanceID      string
	UprojectPath    string
	LastHeartbeatAt string
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

func explicitProjectDirSet() bool {
	return configuredProjectDirEnv() != ""
}

func configuredProjectDirEnv() string {
	env := strings.TrimSpace(os.Getenv("NEOSTACK_PROJECT_DIR"))
	if env == "" || strings.Contains(env, "${user_config.") {
		return ""
	}
	return env
}

func findProjectDir() (string, error) {
	if env := configuredProjectDirEnv(); env != "" {
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
		"Run claude/codex from inside your UE project directory, set NEOSTACK_PROJECT_DIR to its absolute path, or keep only one NeoStackAI-enabled editor open for automatic desktop discovery.",
	)
}

func globalRegistryPath() string {
	if override := os.Getenv("NEOSTACK_RUNTIME_REGISTRY"); override != "" {
		return override
	}

	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "NeoStackAI", "runtimes.json")
		}
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".neostack", "runtimes.json")
	}

	return filepath.Join(".", "runtimes.json")
}

func discover() (*discoveredEditor, error) {
	projectDir, projectErr := findProjectDir()
	if projectErr == nil {
		if editor, err := discoverProject(projectDir); err == nil {
			return editor, nil
		} else {
			if explicitProjectDirSet() {
				return nil, err
			}
			if editor, registryErr := discoverFromRegistry(projectDir); registryErr == nil {
				return editor, nil
			}
			return nil, err
		}
	}

	if explicitProjectDirSet() {
		return nil, projectErr
	}

	return discoverFromRegistry("")
}

func discoverProject(projectDir string) (*discoveredEditor, error) {
	runtimePath := filepath.Join(projectDir, "Saved", "NeoStackAI", "runtime.json")
	rt, err := readRuntimeFile(runtimePath)
	if err != nil {
		return nil, err
	}
	return editorFromRuntime(rt)
}

func readRuntimeFile(runtimePath string) (runtimeFile, error) {
	data, err := os.ReadFile(runtimePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return runtimeFile{}, newDiscoveryError(
				fmt.Sprintf("No NeoStackAI runtime file at %s.", runtimePath),
				"Open the Unreal editor for this project with the NeoStackAI plugin enabled.",
			)
		}
		return runtimeFile{}, err
	}

	var rt runtimeFile
	if err := json.Unmarshal(data, &rt); err != nil {
		return runtimeFile{}, newDiscoveryError(
			fmt.Sprintf("runtime.json is not valid JSON: %v", err),
			"",
		)
	}
	return rt, nil
}

func readRegistryFile() (registryFile, error) {
	registryPath := globalRegistryPath()
	data, err := os.ReadFile(registryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return registryFile{}, newDiscoveryError(
				fmt.Sprintf("No NeoStackAI runtime registry at %s.", registryPath),
				"Open an Unreal editor with the NeoStackAI plugin enabled, or run from inside the project directory.",
			)
		}
		return registryFile{}, err
	}

	var registry registryFile
	if err := json.Unmarshal(data, &registry); err != nil {
		return registryFile{}, newDiscoveryError(
			fmt.Sprintf("NeoStackAI runtime registry is not valid JSON: %v", err),
			"",
		)
	}

	if registry.SchemaVersion != 1 {
		return registryFile{}, newDiscoveryError(
			fmt.Sprintf("NeoStackAI runtime registry schemaVersion is %d, expected 1.", registry.SchemaVersion),
			"Update the NeoStackAI plugin in your editor to a version that writes the multi-project runtime registry.",
		)
	}

	return registry, nil
}

func discoverFromRegistry(preferredProjectDir string) (*discoveredEditor, error) {
	editors, err := activeEditorsFromRegistry()
	if err != nil {
		return nil, err
	}

	if preferredProjectDir != "" {
		matches := make([]discoveredEditor, 0, 1)
		for _, editor := range editors {
			if samePath(editor.ProjectPath, preferredProjectDir) {
				matches = append(matches, editor)
			}
		}
		return chooseDiscoveredEditor(matches, "Multiple active NeoStackAI editors match this project.")
	}

	return chooseDiscoveredEditor(editors, "Multiple active NeoStackAI editors are running.")
}

func activeEditorsFromRegistry() ([]discoveredEditor, error) {
	registry, err := readRegistryFile()
	if err != nil {
		return nil, err
	}

	editors := make([]discoveredEditor, 0, len(registry.Runtimes))
	for _, rt := range registry.Runtimes {
		editor, err := editorFromRuntime(rt)
		if err == nil {
			editors = append(editors, *editor)
		}
	}

	if len(editors) == 0 {
		return nil, newDiscoveryError(
			fmt.Sprintf("No active NeoStackAI editors were found in %s.", globalRegistryPath()),
			"Open an Unreal editor with NeoStackAI enabled and wait a few seconds for discovery to update.",
		)
	}

	return editors, nil
}

func chooseDiscoveredEditor(editors []discoveredEditor, multiMsg string) (*discoveredEditor, error) {
	switch len(editors) {
	case 0:
		return nil, newDiscoveryError(
			"No active NeoStackAI editor matches the selected project.",
			"Open that project in Unreal Editor, or set NEOSTACK_PROJECT_DIR to the project you want this connector to use.",
		)
	case 1:
		return &editors[0], nil
	default:
		return nil, newDiscoveryError(
			multiMsg+"\n\n"+formatEditorChoices(editors),
			"Set NEOSTACK_PROJECT_DIR to the absolute project directory for the editor you want this connector to use.",
		)
	}
}

func editorFromRuntime(rt runtimeFile) (*discoveredEditor, error) {
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
				URL:             s.URL,
				ProjectName:     rt.ProjectName,
				ProjectPath:     rt.ProjectPath,
				InstanceID:      rt.InstanceID,
				UprojectPath:    rt.UprojectPath,
				LastHeartbeatAt: rt.LastHeartbeatAt,
			}, nil
		}
	}

	return nil, newDiscoveryError(
		"runtime.json has no MCP server with type='http'.",
		"This proxy currently only supports HTTP transport.",
	)
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil {
		a = absA
	}
	if errB == nil {
		b = absB
	}
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func formatEditorChoices(editors []discoveredEditor) string {
	lines := make([]string, 0, len(editors)+1)
	lines = append(lines, "Active editors:")
	for _, editor := range editors {
		projectName := editor.ProjectName
		if projectName == "" {
			projectName = "(unnamed project)"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", projectName, editor.ProjectPath))
	}
	return strings.Join(lines, "\n")
}
