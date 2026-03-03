package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hengyunabc/mcp2cli/internal/shellpath"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestRunDynamicHelpAndInvocation(t *testing.T) {
	serverURL, shutdown := startTestMCPServer(t, "good-token")
	defer shutdown()

	cfgPath := writeConfig(t, fmt.Sprintf(`{
  "name": "weather",
  "server": {
    "url": %q,
    "transport": "streamable_http",
    "timeout_seconds": 10
  },
  "auth": {
    "type": "bearer",
    "token": "good-token"
  },
  "cli": {
    "description": "Weather MCP CLI",
    "include_return_schema_in_help": true,
    "pretty_json": false
  }
}`, serverURL))

	t.Run("root help shows dynamic tools", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), "mcp2cli", []string{"--config", cfgPath, "--help"}, strings.NewReader(""), &stdout, &stderr, "test-version")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		output := stdout.String()
		if !strings.Contains(output, "cityInfo") {
			t.Fatalf("root help does not include cityInfo tool:\n%s", output)
		}
		if !strings.Contains(output, "Discovered tools") {
			t.Fatalf("root help does not include discovered tools section:\n%s", output)
		}
	})

	t.Run("tool help includes params and return schema", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), "mcp2cli", []string{"--config", cfgPath, "cityInfo", "--help"}, strings.NewReader(""), &stdout, &stderr, "test-version")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}
		output := stdout.String()
		if !strings.Contains(output, "Parameters:") {
			t.Fatalf("tool help missing Parameters section:\n%s", output)
		}
		if !strings.Contains(output, "--name") {
			t.Fatalf("tool help missing --name parameter:\n%s", output)
		}
		if !strings.Contains(output, "Return Schema:") {
			t.Fatalf("tool help missing Return Schema section:\n%s", output)
		}
		if !strings.Contains(output, "\"tempC\"") {
			t.Fatalf("tool help missing output schema field tempC:\n%s", output)
		}
	})

	t.Run("tool invocation returns json", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), "mcp2cli", []string{"--config", cfgPath, "cityInfo", "--name", "hk"}, strings.NewReader(""), &stdout, &stderr, "test-version")
		if code != 0 {
			t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
		}

		var payload map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
			t.Fatalf("invalid JSON output: %v, raw: %s", err, stdout.String())
		}
		sc, ok := payload["structuredContent"].(map[string]any)
		if !ok {
			t.Fatalf("structuredContent missing from output: %s", stdout.String())
		}
		if got, want := sc["city"], "hk"; got != want {
			t.Fatalf("city = %#v, want %#v", got, want)
		}
	})

	t.Run("missing required parameter returns code 2", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := Run(context.Background(), "mcp2cli", []string{"--config", cfgPath, "cityInfo"}, strings.NewReader(""), &stdout, &stderr, "test-version")
		if code != 2 {
			t.Fatalf("exit code = %d, want 2, stderr = %s", code, stderr.String())
		}
		if !strings.Contains(stderr.String(), "--name") {
			t.Fatalf("missing required parameter error not clear: %s", stderr.String())
		}
	})
}

func TestRunAuthFailureReturnsConnectionCode(t *testing.T) {
	serverURL, shutdown := startTestMCPServer(t, "good-token")
	defer shutdown()

	cfgPath := writeConfig(t, fmt.Sprintf(`{
  "name": "weather",
  "server": {
    "url": %q,
    "transport": "streamable_http",
    "timeout_seconds": 10
  },
  "auth": {
    "type": "bearer",
    "token": "wrong-token"
  }
}`, serverURL))

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), "mcp2cli", []string{"--config", cfgPath, "--help"}, strings.NewReader(""), &stdout, &stderr, "test-version")
	if code != 3 {
		t.Fatalf("exit code = %d, want 3, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "unauthorized") {
		t.Fatalf("auth failure stderr should mention unauthorized, got: %s", stderr.String())
	}
}

func TestRunInstallListRemoveAndWrapperFallback(t *testing.T) {
	requireSymlinkSupport(t)

	serverURL, shutdown := startTestMCPServer(t, "good-token")
	defer shutdown()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("PATH", "/usr/bin")

	var stdout, stderr bytes.Buffer

	code := Run(
		context.Background(),
		"mcp2cli",
		[]string{"install", "weather", "--url", serverURL, "--token", "good-token", "--yes"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		"test-version",
	)
	if code != 0 {
		t.Fatalf("install exit code = %d, stderr = %s", code, stderr.String())
	}

	configPath := filepath.Join(homeDir, ".mcp2cli", "configs", "weather.json")
	wrapperPath := filepath.Join(homeDir, ".mcp2cli", "bin", "weather")
	rcPath := filepath.Join(homeDir, ".bashrc")

	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file not found: %v", err)
	}
	if _, err := os.Stat(wrapperPath); err != nil {
		t.Fatalf("wrapper file not found: %v", err)
	}
	rcContent, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file error: %v", err)
	}
	if !strings.Contains(string(rcContent), shellpath.PathExportLine) {
		t.Fatalf("rc file does not contain PATH export line: %s", string(rcContent))
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(
		context.Background(),
		wrapperPath,
		[]string{"--help"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		"test-version",
	)
	if code != 0 {
		t.Fatalf("wrapper help exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "cityInfo") {
		t.Fatalf("wrapper help output should include dynamic tool cityInfo, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(
		context.Background(),
		"mcp2cli",
		[]string{"list"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		"test-version",
	)
	if code != 0 {
		t.Fatalf("list exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "weather") || !strings.Contains(stdout.String(), serverURL) {
		t.Fatalf("list output missing expected values: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(
		context.Background(),
		"mcp2cli",
		[]string{"install", "weather", "--url", serverURL, "--token", "good-token"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		"test-version",
	)
	if code != 5 {
		t.Fatalf("reinstall exit code = %d, want 5, stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(
		context.Background(),
		"mcp2cli",
		[]string{"remove", "weather"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		"test-version",
	)
	if code != 0 {
		t.Fatalf("remove exit code = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("config should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(wrapperPath); !os.IsNotExist(err) {
		t.Fatalf("wrapper should be removed, stat err = %v", err)
	}
}

func TestRunWrapperUsesInvokedNameWhenConfigNameMissing(t *testing.T) {
	requireSymlinkSupport(t)

	serverURL, shutdown := startTestMCPServer(t, "")
	defer shutdown()

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	cfgPath := filepath.Join(homeDir, ".mcp2cli", "configs", "weather.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(fmt.Sprintf(`{
  "server": {
    "url": %q
  }
}`, serverURL)), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(
		context.Background(),
		"/usr/local/bin/weather",
		[]string{"--help"},
		strings.NewReader(""),
		&stdout,
		&stderr,
		"test-version",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage:\n  weather") {
		t.Fatalf("help output should use invoked name weather, got: %s", stdout.String())
	}
}

func startTestMCPServer(t *testing.T, token string) (string, func()) {
	t.Helper()

	mcpServer := server.NewMCPServer("weather-server", "1.0.0")
	mcpServer.AddTool(mcp.Tool{
		Name:        "cityInfo",
		Description: "Get weather information for a city",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "City name",
				},
				"unit": map[string]any{
					"type":    "string",
					"enum":    []any{"c", "f"},
					"default": "c",
				},
			},
			Required: []string{"name"},
		},
		OutputSchema: mcp.ToolOutputSchema{
			Type: "object",
			Properties: map[string]any{
				"city": map[string]any{
					"type": "string",
				},
				"weather": map[string]any{
					"type": "string",
				},
				"tempC": map[string]any{
					"type": "number",
				},
			},
			Required: []string{"city", "weather", "tempC"},
		},
	}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("invalid parameter", err), nil
		}

		unit := request.GetString("unit", "c")
		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Weather for %s", name),
				},
			},
			StructuredContent: map[string]any{
				"city":    name,
				"weather": "sunny",
				"tempC":   26.5,
				"unit":    unit,
			},
		}
		return result, nil
	})

	mcpHandler := server.NewStreamableHTTPServer(mcpServer)
	mux := http.NewServeMux()
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth := r.Header.Get("Authorization")
		wantAuth := "Bearer " + token
		if token != "" && gotAuth != wantAuth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mcpHandler.ServeHTTP(w, r)
	}))

	testServer := httptest.NewServer(mux)
	return testServer.URL + "/mcp", testServer.Close
}

func requireSymlinkSupport(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile target error: %v", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
