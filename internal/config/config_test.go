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
		path, err := ExtractConfigPath([]string{"--config", "a.json", "tools"})
		if err != nil {
			t.Fatalf("ExtractConfigPath error: %v", err)
		}
		if got, want := path, "a.json"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})

	t.Run("long flag equals", func(t *testing.T) {
		path, err := ExtractConfigPath([]string{"--config=b.json"})
		if err != nil {
			t.Fatalf("ExtractConfigPath error: %v", err)
		}
		if got, want := path, "b.json"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})

	t.Run("fallback env", func(t *testing.T) {
		t.Setenv("MCP2CLI_CONFIG", "from-env.json")
		path, err := ExtractConfigPath([]string{"tools"})
		if err != nil {
			t.Fatalf("ExtractConfigPath error: %v", err)
		}
		if got, want := path, "from-env.json"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})
}
