package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/michaelpeterswa/mcp-wsdot/internal/config"
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

// StartServer builds the MCP server and serves it over the configured
// transport. It blocks until the transport stops or ctx is cancelled; for the
// HTTP transports, cancelling ctx triggers a graceful drain bounded by
// c.ShutdownTimeout.
func StartServer(ctx context.Context, c *config.Config, opts ...MCPServerOption) error {
	s := newMCPServer(c.ServerName, opts...)

	switch c.ParsedTransport {
	case config.TransportStdio:
		return server.NewStdioServer(s).Listen(ctx, os.Stdin, os.Stdout)
	case config.TransportStreamableHTTP:
		return startStreamableHTTP(ctx, c, s)
	case config.TransportSSE:
		return startSSE(ctx, c, s)
	default:
		return fmt.Errorf("unhandled transport type: %q", c.ParsedTransport)
	}
}

// startStreamableHTTP serves the streamable HTTP transport, the transport to
// use when hosting this server remotely.
func startStreamableHTTP(ctx context.Context, c *config.Config, s *server.MCPServer) error {
	streamable := server.NewStreamableHTTPServer(s,
		server.WithEndpointPath(c.HTTPPath),
		server.WithStateLess(c.Stateless),
		server.WithHeartbeatInterval(c.HeartbeatInterval),
		server.WithStreamableHTTPLogger(slog.Default()),
	)

	mux := newServeMux(c, c.HTTPPath, streamable)

	return serve(ctx, c, mux, streamable.Shutdown)
}

// startSSE serves the deprecated HTTP+SSE transport. The MCP specification
// superseded it with streamable HTTP in March 2025; it remains here only for
// clients that have not migrated.
func startSSE(ctx context.Context, c *config.Config, s *server.MCPServer) error {
	slog.Warn("the sse transport is deprecated by the mcp specification, prefer TRANSPORT=streamablehttp")

	sse := server.NewSSEServer(s,
		server.WithKeepAlive(c.HeartbeatInterval > 0),
		server.WithKeepAliveInterval(c.HeartbeatInterval),
	)

	// SSEServer routes internally between its own /sse and /message endpoints,
	// so it is mounted as the catch-all; the health patterns registered by
	// newServeMux are more specific and still win.
	mux := newServeMux(c, "/", sse)

	return serve(ctx, c, mux, sse.Shutdown)
}
