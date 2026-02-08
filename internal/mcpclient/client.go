package mcpclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hengyunabc/mcp2cli/internal/config"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type Client struct {
	rawClient *client.Client
	tools     []mcp.Tool
}

func Connect(ctx context.Context, cfg config.Config, version string) (*Client, error) {
	options := []transport.StreamableHTTPCOption{
		transport.WithHTTPTimeout(time.Duration(cfg.Server.TimeoutSeconds) * time.Second),
	}

	headers := map[string]string{}
	if strings.TrimSpace(cfg.Auth.Type) == "bearer" && strings.TrimSpace(cfg.Auth.Token) != "" {
		headers["Authorization"] = "Bearer " + strings.TrimSpace(cfg.Auth.Token)
	}
	if len(headers) > 0 {
		options = append(options, transport.WithHTTPHeaders(headers))
	}

	rawClient, err := client.NewStreamableHttpClient(cfg.Server.URL, options...)
	if err != nil {
		return nil, fmt.Errorf("create MCP streamable HTTP client: %w", err)
	}

	if err := rawClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("start MCP client transport: %w", err)
	}

	initReq := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			Capabilities:    mcp.ClientCapabilities{},
			ClientInfo: mcp.Implementation{
				Name:    "mcp2cli",
				Version: version,
			},
		},
	}
	if _, err := rawClient.Initialize(ctx, initReq); err != nil {
		_ = rawClient.Close()
		return nil, fmt.Errorf("initialize MCP session: %w", err)
	}

	toolResult, err := rawClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		_ = rawClient.Close()
		return nil, fmt.Errorf("fetch MCP tools list: %w", err)
	}

	return &Client{
		rawClient: rawClient,
		tools:     toolResult.Tools,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.rawClient == nil {
		return nil
	}
	return c.rawClient.Close()
}

func (c *Client) Tools() []mcp.Tool {
	if c == nil {
		return nil
	}
	out := make([]mcp.Tool, len(c.tools))
	copy(out, c.tools)
	return out
}

func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	if c == nil || c.rawClient == nil {
		return nil, fmt.Errorf("MCP client is not initialized")
	}
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	}
	result, err := c.rawClient.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("call MCP tool %q: %w", toolName, err)
	}
	return result, nil
}
