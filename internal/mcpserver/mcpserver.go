package mcpserver

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/michaelpeterswa/go-mcp-template/internal/config"
)

type MCPServer struct {
	serverOptions []server.ServerOption
	tools         []Tool
}

type MCPServerOption func(*MCPServer)

func WithServerOptions(opts ...server.ServerOption) MCPServerOption {
	return func(m *MCPServer) {
		m.serverOptions = opts
	}
}

func WithTools(tools []Tool) MCPServerOption {
	return func(m *MCPServer) {
		m.tools = tools
	}
}

type Tool struct {
	t  mcp.Tool
	th server.ToolHandlerFunc
}

func NewTool(t mcp.Tool, th server.ToolHandlerFunc) Tool {
	return Tool{
		t:  t,
		th: th,
	}
}

func newMCPServer(name string, opts ...MCPServerOption) *server.MCPServer {
	var mcpServer MCPServer

	for _, opt := range opts {
		opt(&mcpServer)
	}

	s := server.NewMCPServer(
		name,
		config.AppVersion,
		mcpServer.serverOptions...,
	)

	for _, tool := range mcpServer.tools {
		s.AddTool(tool.t, tool.th)
	}

	return s
}

func StartServer(ctx context.Context, c *config.Config, opts ...MCPServerOption) error {
	switch c.Transport {
	case "stdio":
		return server.NewStdioServer(newMCPServer(c.ServerName, opts...)).Listen(context.Background(), os.Stdin, os.Stdout)
	case "sse":
		return server.NewSSEServer(newMCPServer(c.ServerName, opts...)).Start(fmt.Sprintf(":%d", c.SSEPort))
	default:
		return fmt.Errorf(
			"invalid transport type: %s, valid types are: stdio, sse",
			c.Transport,
		)
	}
}
