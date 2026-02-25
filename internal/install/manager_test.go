package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerInstallListRemove(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mcp2cli")
	manager, err := NewManager(baseDir)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	result, err := manager.Install(InstallOptions{
		Name:  "weather",
		URL:   "https://example.com/mcp",
		Token: "test-token",
	})
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}

	if _, err := os.Stat(result.ConfigPath); err != nil {
		t.Fatalf("config file missing: %v", err)
	}
	if _, err := os.Stat(result.WrapperPath); err != nil {
		t.Fatalf("wrapper file missing: %v", err)
	}

	wrapperContent, err := os.ReadFile(result.WrapperPath)
	if err != nil {
		t.Fatalf("ReadFile wrapper error: %v", err)
	}
	if !strings.Contains(string(wrapperContent), `$HOME/.mcp2cli/configs/weather.json`) {
		t.Fatalf("unexpected wrapper content: %s", string(wrapperContent))
	}

	entries, err := manager.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries length = %d, want 1", len(entries))
	}
	if got, want := entries[0].Name, "weather"; got != want {
		t.Fatalf("entry name = %q, want %q", got, want)
	}

	removeResult, err := manager.Remove("weather", false)
	if err != nil {
		t.Fatalf("Remove error: %v", err)
	}
	if !removeResult.WrapperRemoved || !removeResult.ConfigRemoved {
		t.Fatalf("remove result = %#v, want wrapper+config removed", removeResult)
	}
}

func TestManagerInstallForce(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mcp2cli")
	manager, err := NewManager(baseDir)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	_, err = manager.Install(InstallOptions{
		Name: "weather",
		URL:  "https://example.com/mcp",
	})
	if err != nil {
		t.Fatalf("first Install error: %v", err)
	}

	_, err = manager.Install(InstallOptions{
		Name: "weather",
		URL:  "https://example.com/mcp",
	})
	if err == nil {
		t.Fatalf("expected second install without --force to fail")
	}

	_, err = manager.Install(InstallOptions{
		Name:  "weather",
		URL:   "https://example.com/mcp",
		Force: true,
	})
	if err != nil {
		t.Fatalf("install with --force error: %v", err)
	}
}

func TestManagerInstallFromConfig(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), ".mcp2cli")
	manager, err := NewManager(baseDir)
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}

	sourceConfig := filepath.Join(t.TempDir(), "source.json")
	err = os.WriteFile(sourceConfig, []byte(`{
  "name": "source",
  "server": {
    "url": "https://mcp.example.com/server",
    "transport": "streamable_http",
    "timeout_seconds": 20
  },
  "auth": {
    "type": "bearer",
    "token": "abc"
  }
}`), 0o644)
	if err != nil {
		t.Fatalf("WriteFile source config error: %v", err)
	}

	result, err := manager.Install(InstallOptions{
		Name:           "aone-project",
		FromConfigPath: sourceConfig,
	})
	if err != nil {
		t.Fatalf("Install from config error: %v", err)
	}

	content, err := os.ReadFile(result.ConfigPath)
	if err != nil {
		t.Fatalf("ReadFile result config error: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `"name": "aone-project"`) {
		t.Fatalf("config should use target name, got: %s", text)
	}
	if !strings.Contains(text, `"url": "https://mcp.example.com/server"`) {
		t.Fatalf("config should preserve source server url, got: %s", text)
	}
}
