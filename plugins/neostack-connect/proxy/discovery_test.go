package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDiscoverFallsBackToSingleRegistryEditor(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NEOSTACK_RUNTIME_REGISTRY", filepath.Join(tmp, "runtimes.json"))
	t.Setenv("NEOSTACK_PROJECT_DIR", "")
	chdir(t, tmp)

	projectDir := filepath.Join(tmp, "Game")
	writeRegistry(t, validRuntime("Game", projectDir, "http://127.0.0.1:7777/mcp"))

	editor, err := discover()
	if err != nil {
		t.Fatalf("discover() error = %v", err)
	}
	if editor.ProjectName != "Game" {
		t.Fatalf("ProjectName = %q, want Game", editor.ProjectName)
	}
	if editor.URL != "http://127.0.0.1:7777/mcp" {
		t.Fatalf("URL = %q", editor.URL)
	}
}

func TestDiscoverMatchesCurrentProjectFromRegistry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NEOSTACK_RUNTIME_REGISTRY", filepath.Join(tmp, "runtimes.json"))
	t.Setenv("NEOSTACK_PROJECT_DIR", "")

	projectA := filepath.Join(tmp, "ProjectA")
	projectB := filepath.Join(tmp, "ProjectB")
	if err := os.MkdirAll(filepath.Join(projectA, "Source"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectA, "ProjectA.uproject"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdir(t, filepath.Join(projectA, "Source"))

	writeRegistry(t,
		validRuntime("ProjectA", projectA, "http://127.0.0.1:7001/mcp"),
		validRuntime("ProjectB", projectB, "http://127.0.0.1:7002/mcp"),
	)

	editor, err := discover()
	if err != nil {
		t.Fatalf("discover() error = %v", err)
	}
	if editor.ProjectName != "ProjectA" {
		t.Fatalf("ProjectName = %q, want ProjectA", editor.ProjectName)
	}
}

func TestDiscoverRejectsAmbiguousRegistryEditors(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NEOSTACK_RUNTIME_REGISTRY", filepath.Join(tmp, "runtimes.json"))
	t.Setenv("NEOSTACK_PROJECT_DIR", "")
	chdir(t, tmp)

	writeRegistry(t,
		validRuntime("ProjectA", filepath.Join(tmp, "ProjectA"), "http://127.0.0.1:7001/mcp"),
		validRuntime("ProjectB", filepath.Join(tmp, "ProjectB"), "http://127.0.0.1:7002/mcp"),
	)

	_, err := discover()
	if err == nil {
		t.Fatal("discover() error = nil, want ambiguous project error")
	}
	if !strings.Contains(err.Error(), "Multiple active NeoStackAI editors") {
		t.Fatalf("error = %q", err.Error())
	}
}

func validRuntime(projectName, projectDir, url string) runtimeFile {
	return runtimeFile{
		SchemaVersion:   2,
		InstanceID:      projectName + "-instance",
		EditorPID:       1234,
		ProjectName:     projectName,
		ProjectPath:     projectDir,
		UprojectPath:    filepath.Join(projectDir, projectName+".uproject"),
		PluginVersion:   "1.0.0",
		EngineVersion:   "5.7.0",
		StartedAt:       time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano),
		LastHeartbeatAt: time.Now().UTC().Format(time.RFC3339Nano),
		MCPRunning:      true,
		MCPServers: []runtimeServer{
			{Name: "unreal-editor", Type: "http", URL: url},
		},
	}
}

func writeRegistry(t *testing.T, runtimes ...runtimeFile) {
	t.Helper()
	path := globalRegistryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(registryFile{
		SchemaVersion: 1,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		Runtimes:      runtimes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldwd)
	})
}
