package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExpandsEnvAndAppliesDefaults(t *testing.T) {
	t.Setenv("TEST_MCP_TOKEN", "token-from-env")

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "weather.json")
	err := os.WriteFile(cfgPath, []byte(`{
  "name": "weather",
  "server": {
    "url": "http://localhost:8080/mcp"
  },
  "auth": {
    "type": "bearer",
    "token": "${TEST_MCP_TOKEN}"
  }
}`), 0o644)
	if err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got, want := cfg.Server.Transport, DefaultTransport; got != want {
		t.Fatalf("transport = %q, want %q", got, want)
	}
	if got, want := cfg.Server.TimeoutSeconds, DefaultTimeoutSeconds; got != want {
		t.Fatalf("timeout = %d, want %d", got, want)
	}
	if got, want := cfg.Auth.Token, "token-from-env"; got != want {
		t.Fatalf("token = %q, want %q", got, want)
	}
}

func TestExtractConfigPath(t *testing.T) {
	t.Run("long flag with value", func(t *testing.T) {
		path, err := ExtractConfigPath([]string{"--config", "a.json", "tools"}, "mcp2cli")
		if err != nil {
			t.Fatalf("ExtractConfigPath error: %v", err)
		}
		if got, want := path, "a.json"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})

	t.Run("long flag equals", func(t *testing.T) {
		path, err := ExtractConfigPath([]string{"--config=b.json"}, "mcp2cli")
		if err != nil {
			t.Fatalf("ExtractConfigPath error: %v", err)
		}
		if got, want := path, "b.json"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})

	t.Run("fallback env", func(t *testing.T) {
		t.Setenv("MCP2CLI_CONFIG", "from-env.json")
		path, err := ExtractConfigPath([]string{"tools"}, "mcp2cli")
		if err != nil {
			t.Fatalf("ExtractConfigPath error: %v", err)
		}
		if got, want := path, "from-env.json"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})

	t.Run("fallback from wrapper command name", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		expectedDir := filepath.Join(homeDir, ".mcp2cli", "configs")
		if err := os.MkdirAll(expectedDir, 0o755); err != nil {
			t.Fatalf("MkdirAll error: %v", err)
		}
		expectedPath := filepath.Join(expectedDir, "weather.json")
		if err := os.WriteFile(expectedPath, []byte(`{"server":{"url":"http://localhost:8080/mcp"}}`), 0o644); err != nil {
			t.Fatalf("WriteFile error: %v", err)
		}

		path, err := ExtractConfigPath([]string{"--help"}, "/usr/local/bin/weather")
		if err != nil {
			t.Fatalf("ExtractConfigPath error: %v", err)
		}
		if got, want := path, expectedPath; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})

	t.Run("mcp2cli command does not use argv0 fallback", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		path, err := ExtractConfigPath([]string{"tools"}, "/usr/local/bin/mcp2cli")
		if err != nil {
			t.Fatalf("ExtractConfigPath error: %v", err)
		}
		if path != "" {
			t.Fatalf("path = %q, want empty", path)
		}
	})
}
