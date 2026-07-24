package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

var (
	AppVersion = "unset"
)

// Transport identifies which MCP transport the server listens on.
type Transport string

const (
	// TransportStdio serves MCP over stdin/stdout, for local clients that
	// launch the binary themselves (Claude Desktop, the MCP Inspector).
	TransportStdio Transport = "stdio"

	// TransportStreamableHTTP serves MCP over the streamable HTTP transport.
	// This is the transport to use when hosting the server remotely.
	TransportStreamableHTTP Transport = "streamablehttp"

	// TransportSSE serves MCP over the deprecated HTTP+SSE transport. It is
	// kept only for older clients; new deployments should use
	// TransportStreamableHTTP.
	TransportSSE Transport = "sse"
)

// ParseTransport normalizes the spellings of a transport name that clients and
// docs use interchangeably ("streamable-http", "streamable_http", "http") onto
// a single canonical Transport.
func ParseTransport(s string) (Transport, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "stdio":
		return TransportStdio, nil
	case "streamablehttp", "streamable-http", "streamable_http", "http":
		return TransportStreamableHTTP, nil
	case "sse":
		return TransportSSE, nil
	default:
		return "", fmt.Errorf(
			"invalid transport type: %q, valid types are: stdio, streamablehttp, sse",
			s,
		)
	}
}

// IsHTTP reports whether the transport listens on a TCP port.
func (t Transport) IsHTTP() bool {
	return t == TransportStreamableHTTP || t == TransportSSE
}

type Config struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"error"`

	MetricsEnabled bool `env:"METRICS_ENABLED" envDefault:"false"`
	MetricsPort    int  `env:"METRICS_PORT" envDefault:"8081"`

	ServerName string `env:"SERVER_NAME" envDefault:"mcp-wsdot"`

	WSDOTAPIKey     string        `env:"WSDOT_API_KEY"`
	WSDOTAPITimeout time.Duration `env:"WSDOT_API_TIMEOUT" envDefault:"5s"`

	Local bool `env:"LOCAL" envDefault:"false"`

	// Transport selects the MCP transport. See ParseTransport for accepted
	// spellings; the canonical value lives in ParsedTransport.
	Transport       string `env:"TRANSPORT" envDefault:"streamablehttp"`
	ParsedTransport Transport

	// HTTPPort is the listen port for the streamablehttp and sse transports.
	HTTPPort int `env:"HTTP_PORT" envDefault:"8080"`

	// HTTPPath is the path the MCP endpoint is served on. Clients point at
	// http://<host>:<HTTPPort><HTTPPath>.
	HTTPPath string `env:"HTTP_PATH" envDefault:"/mcp"`

	// AuthToken, when set, requires every MCP request to carry
	// "Authorization: Bearer <AuthToken>". Health endpoints are never
	// authenticated so Kubernetes probes still work. Empty disables auth.
	AuthToken string `env:"MCP_AUTH_TOKEN"`

	// Stateless disables server-side session tracking so requests can be load
	// balanced across replicas without sticky sessions.
	Stateless bool `env:"MCP_STATELESS" envDefault:"false"`

	// HeartbeatInterval keeps idle SSE streams alive through proxies and
	// ingress controllers that drop quiet connections. Zero disables it.
	HeartbeatInterval time.Duration `env:"MCP_HEARTBEAT_INTERVAL" envDefault:"30s"`

	// ShutdownTimeout bounds how long in-flight requests get to drain after a
	// SIGTERM before the process exits.
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"15s"`

	TracingEnabled    bool    `env:"TRACING_ENABLED" envDefault:"false"`
	TracingSampleRate float64 `env:"TRACING_SAMPLERATE" envDefault:"0.01"`
	TracingService    string  `env:"TRACING_SERVICE" envDefault:"mcp-wsdot"`
	TracingVersion    string  `env:"TRACING_VERSION"`
}

func NewConfig() (*Config, error) {
	var cfg Config

	err := env.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// SSE_PORT was the only listen port before the streamablehttp transport
	// existed. Honor it when HTTP_PORT has not been set explicitly.
	if _, ok := os.LookupEnv("HTTP_PORT"); !ok {
		var legacy struct {
			Port int `env:"SSE_PORT" envDefault:"8080"`
		}
		if err := env.Parse(&legacy); err != nil {
			return nil, fmt.Errorf("failed to parse legacy SSE_PORT: %w", err)
		}
		cfg.HTTPPort = legacy.Port
	}

	cfg.ParsedTransport, err = ParseTransport(cfg.Transport)
	if err != nil {
		return nil, err
	}

	if cfg.ParsedTransport.IsHTTP() {
		if !strings.HasPrefix(cfg.HTTPPath, "/") {
			return nil, fmt.Errorf("HTTP_PATH must begin with '/', got %q", cfg.HTTPPath)
		}
		if cfg.HTTPPort < 1 || cfg.HTTPPort > 65535 {
			return nil, fmt.Errorf("HTTP_PORT must be between 1 and 65535, got %d", cfg.HTTPPort)
		}
		if cfg.MetricsEnabled && cfg.MetricsPort == cfg.HTTPPort {
			return nil, fmt.Errorf("HTTP_PORT and METRICS_PORT must differ, both are %d", cfg.HTTPPort)
		}
	}

	return &cfg, nil
}
