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
		code := Run(context.Background(), []string{"--config", cfgPath, "--help"}, &stdout, &stderr, "test-version")
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
		code := Run(context.Background(), []string{"--config", cfgPath, "cityInfo", "--help"}, &stdout, &stderr, "test-version")
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
		code := Run(context.Background(), []string{"--config", cfgPath, "cityInfo", "--name", "hk"}, &stdout, &stderr, "test-version")
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
		code := Run(context.Background(), []string{"--config", cfgPath, "cityInfo"}, &stdout, &stderr, "test-version")
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
	code := Run(context.Background(), []string{"--config", cfgPath, "--help"}, &stdout, &stderr, "test-version")
	if code != 3 {
		t.Fatalf("exit code = %d, want 3, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "unauthorized") {
		t.Fatalf("auth failure stderr should mention unauthorized, got: %s", stderr.String())
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

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
