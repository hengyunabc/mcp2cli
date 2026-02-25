package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/hengyunabc/mcp2cli/internal/apperr"
	"github.com/hengyunabc/mcp2cli/internal/config"
	"github.com/hengyunabc/mcp2cli/internal/mcpclient"
	"github.com/hengyunabc/mcp2cli/internal/render"
	"github.com/hengyunabc/mcp2cli/internal/schema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"
)

type Runtime struct {
	ConfigPath string
	Config     config.Config
	Client     *mcpclient.Client
	Stdout     io.Writer
	Stderr     io.Writer
	Version    string
}

func NewRootCommand(rt Runtime) (*cobra.Command, error) {
	commandName := sanitizeCommandName(rt.Config.Name)
	root := &cobra.Command{
		Use:   commandName,
		Short: rootShort(rt.Config),
		Long:  rootLong(rt.Config, rt.Client.Tools()),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(rt.Stdout)
	root.SetErr(rt.Stderr)

	root.PersistentFlags().StringP("config", "c", rt.ConfigPath, "Path to config file")

	root.AddCommand(newVersionCommand(rt))
	root.AddCommand(newToolsCommand(rt.Client.Tools(), rt.Stdout))

	toolNames := map[string]struct{}{}
	for _, tool := range rt.Client.Tools() {
		if _, exists := toolNames[tool.Name]; exists {
			return nil, fmt.Errorf("duplicate tool name detected: %q", tool.Name)
		}
		toolNames[tool.Name] = struct{}{}

		toolCmd, err := newToolCommand(rt, tool)
		if err != nil {
			return nil, err
		}
		root.AddCommand(toolCmd)
	}

	return root, nil
}

func newVersionCommand(rt Runtime) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(rt.Stdout, rt.Version)
			return err
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(rt.Stdout)
	cmd.SetErr(rt.Stderr)
	return cmd
}

func newToolsCommand(tools []mcp.Tool, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "List discovered MCP tools",
		Long:  "List tool names discovered from the configured MCP server.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if len(tools) == 0 {
				_, err := fmt.Fprintln(stdout, "No tools discovered.")
				return err
			}
			for _, tool := range tools {
				if strings.TrimSpace(tool.Description) == "" {
					if _, err := fmt.Fprintf(stdout, "%s\n", tool.Name); err != nil {
						return err
					}
					continue
				}
				if _, err := fmt.Fprintf(stdout, "%s: %s\n", tool.Name, oneLine(tool.Description)); err != nil {
					return err
				}
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	return cmd
}

func newToolCommand(rt Runtime, tool mcp.Tool) (*cobra.Command, error) {
	spec, err := schema.ParseToolInputSchema(tool)
	if err != nil {
		return nil, fmt.Errorf("build schema for tool %q: %w", tool.Name, err)
	}

	cmd := &cobra.Command{
		Use:   tool.Name,
		Short: toolShort(tool),
		Long:  toolLong(tool, spec, config.IncludeReturnSchemaInHelp(rt.Config)),
		Example: toolExample(
			sanitizeCommandName(rt.Config.Name),
			tool,
			spec,
		),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			callArgs, err := schema.BuildArguments(cmd, spec)
			if err != nil {
				return apperr.Wrap(apperr.CodeValidation, err, "invalid parameters for tool %q", tool.Name)
			}
			result, err := rt.Client.CallTool(cmd.Context(), tool.Name, callArgs)
			if err != nil {
				return apperr.Wrap(apperr.CodeToolExecution, err, "tool %q call failed", tool.Name)
			}
			if err := render.JSON(rt.Stdout, result, rt.Config.CLI.PrettyJSON); err != nil {
				return apperr.Wrap(apperr.CodeInternal, err, "render tool result")
			}
			if result.IsError {
				return apperr.New(apperr.CodeToolExecution, fmt.Sprintf("tool %q returned isError=true", tool.Name))
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(rt.Stdout)
	cmd.SetErr(rt.Stderr)

	if err := schema.RegisterFlags(cmd, spec); err != nil {
		return nil, fmt.Errorf("register flags for tool %q: %w", tool.Name, err)
	}

	return cmd, nil
}

func sanitizeCommandName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "mcp2cli"
	}
	return trimmed
}

func rootShort(cfg config.Config) string {
	if strings.TrimSpace(cfg.CLI.Description) != "" {
		return cfg.CLI.Description
	}
	return "MCP to CLI bridge"
}

func rootLong(cfg config.Config, tools []mcp.Tool) string {
	parts := []string{
		rootShort(cfg),
		"",
		"Server:",
		fmt.Sprintf("  name: %s", sanitizeCommandName(cfg.Name)),
		fmt.Sprintf("  url: %s", cfg.Server.URL),
		fmt.Sprintf("  transport: %s", cfg.Server.Transport),
		fmt.Sprintf("  auth: %s", authTypeOrNone(cfg.Auth.Type)),
		"",
		"Discovered tools:",
	}
	if len(tools) == 0 {
		parts = append(parts, "  (none)")
	} else {
		for _, tool := range tools {
			desc := oneLine(tool.Description)
			if desc == "" {
				desc = "No description"
			}
			parts = append(parts, fmt.Sprintf("  - %s: %s", tool.Name, desc))
		}
	}

	commandName := sanitizeCommandName(cfg.Name)
	parts = append(parts, "", "Examples:")
	parts = append(parts, fmt.Sprintf("  %s tools", commandName))
	parts = append(parts, fmt.Sprintf("  %s <tool> --help", commandName))
	parts = append(parts, fmt.Sprintf("  %s cityInfo --name hk", commandName))

	return strings.Join(parts, "\n")
}

func toolShort(tool mcp.Tool) string {
	if strings.TrimSpace(tool.Description) == "" {
		return fmt.Sprintf("Run MCP tool %s", tool.Name)
	}
	return oneLine(tool.Description)
}

func toolLong(tool mcp.Tool, spec schema.Spec, includeReturnSchema bool) string {
	parts := []string{
		fmt.Sprintf("Tool: %s", tool.Name),
	}
	if strings.TrimSpace(tool.Description) != "" {
		parts = append(parts, "", tool.Description)
	}

	parameterLines := schema.ParameterLines(spec)
	parts = append(parts, "", "Parameters:")
	if len(parameterLines) == 0 {
		parts = append(parts, "  (none)")
	} else {
		for _, line := range parameterLines {
			parts = append(parts, "  - "+line)
		}
	}

	parts = append(parts, "", "Input Schema:")
	parts = append(parts, indentText(render.PrettyJSONString(schema.InputSchemaMap(tool)), "  "))

	if includeReturnSchema {
		parts = append(parts, "", "Return Schema:")
		outputSchema, ok := schema.OutputSchemaMap(tool)
		if !ok {
			parts = append(parts, "  (server did not provide output schema)")
		} else {
			parts = append(parts, indentText(render.PrettyJSONString(outputSchema), "  "))
		}
	}

	return strings.Join(parts, "\n")
}

func toolExample(commandName string, tool mcp.Tool, spec schema.Spec) string {
	samples := make([]string, 0, 2)
	requiredField := firstRequiredField(spec.Fields)
	if requiredField != nil {
		switch requiredField.Kind {
		case schema.KindString:
			samples = append(samples, fmt.Sprintf("%s %s --%s example", commandName, tool.Name, requiredField.FlagName))
		case schema.KindInteger:
			samples = append(samples, fmt.Sprintf("%s %s --%s 1", commandName, tool.Name, requiredField.FlagName))
		case schema.KindNumber:
			samples = append(samples, fmt.Sprintf("%s %s --%s 1.5", commandName, tool.Name, requiredField.FlagName))
		case schema.KindBoolean:
			samples = append(samples, fmt.Sprintf("%s %s --%s=true", commandName, tool.Name, requiredField.FlagName))
		case schema.KindObject, schema.KindArrayObject:
			samples = append(samples, fmt.Sprintf("%s %s --%s '{\"key\":\"value\"}'", commandName, tool.Name, requiredField.FlagName))
		default:
			samples = append(samples, fmt.Sprintf("%s %s --%s value", commandName, tool.Name, requiredField.FlagName))
		}
	}
	samples = append(samples, fmt.Sprintf("%s %s --help", commandName, tool.Name))
	return strings.Join(samples, "\n")
}

func firstRequiredField(fields []schema.Field) *schema.Field {
	for i := range fields {
		if fields[i].Required {
			return &fields[i]
		}
	}
	return nil
}

func authTypeOrNone(authType string) string {
	if strings.TrimSpace(authType) == "" {
		return "none"
	}
	return authType
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func indentText(text string, indent string) string {
	if text == "" {
		return indent
	}
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}

func ToolNames(tools []mcp.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}
