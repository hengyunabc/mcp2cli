package schema

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
)

func TestBuildArgumentsStrictValidation(t *testing.T) {
	tool := mcp.Tool{
		Name: "cityInfo",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "City name",
				},
				"days": map[string]any{
					"type":    "integer",
					"default": 1,
				},
				"units": map[string]any{
					"type": "string",
					"enum": []any{"c", "f"},
				},
			},
			Required: []string{"name"},
		},
	}

	spec, err := ParseToolInputSchema(tool)
	if err != nil {
		t.Fatalf("ParseToolInputSchema error: %v", err)
	}

	t.Run("missing required field", func(t *testing.T) {
		cmd := &cobra.Command{Use: "cityInfo"}
		if err := RegisterFlags(cmd, spec); err != nil {
			t.Fatalf("RegisterFlags error: %v", err)
		}
		if err := cmd.ParseFlags([]string{"--units", "c"}); err != nil {
			t.Fatalf("ParseFlags error: %v", err)
		}
		_, err := BuildArguments(cmd, spec)
		if err == nil {
			t.Fatalf("expected missing required error")
		}
		if !strings.Contains(err.Error(), "--name") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("enum validation", func(t *testing.T) {
		cmd := &cobra.Command{Use: "cityInfo"}
		if err := RegisterFlags(cmd, spec); err != nil {
			t.Fatalf("RegisterFlags error: %v", err)
		}
		if err := cmd.ParseFlags([]string{"--name", "hk", "--units", "k"}); err != nil {
			t.Fatalf("ParseFlags error: %v", err)
		}
		_, err := BuildArguments(cmd, spec)
		if err == nil {
			t.Fatalf("expected enum validation error")
		}
		if !strings.Contains(err.Error(), "enum") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		cmd := &cobra.Command{Use: "cityInfo"}
		if err := RegisterFlags(cmd, spec); err != nil {
			t.Fatalf("RegisterFlags error: %v", err)
		}
		if err := cmd.ParseFlags([]string{"--name", "hk", "--units", "c"}); err != nil {
			t.Fatalf("ParseFlags error: %v", err)
		}
		args, err := BuildArguments(cmd, spec)
		if err != nil {
			t.Fatalf("BuildArguments error: %v", err)
		}
		if got, want := args["name"], "hk"; got != want {
			t.Fatalf("name = %#v, want %#v", got, want)
		}
		// default should be included when not explicitly set.
		if got, want := args["days"], float64(1); got != want {
			// json defaults may preserve int or float depending source;
			// accept both numeric representations.
			if got != 1 {
				t.Fatalf("days = %#v, want %#v", got, want)
			}
		}
		if got, want := args["units"], "c"; got != want {
			t.Fatalf("units = %#v, want %#v", got, want)
		}
	})
}
