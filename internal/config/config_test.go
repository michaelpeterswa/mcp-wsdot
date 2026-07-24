package config

import (
	"os"
	"testing"
)

func TestParseTransport(t *testing.T) {
	tests := []struct {
		in      string
		want    Transport
		wantErr bool
	}{
		{in: "stdio", want: TransportStdio},
		{in: "sse", want: TransportSSE},
		{in: "streamablehttp", want: TransportStreamableHTTP},
		{in: "streamable-http", want: TransportStreamableHTTP},
		{in: "streamable_http", want: TransportStreamableHTTP},
		{in: "http", want: TransportStreamableHTTP},
		{in: "  Streamable-HTTP  ", want: TransportStreamableHTTP},
		{in: "STDIO", want: TransportStdio},
		{in: "", wantErr: true},
		{in: "grpc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseTransport(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseTransport(%q) = %q, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTransport(%q) returned unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParseTransport(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTransportIsHTTP(t *testing.T) {
	if TransportStdio.IsHTTP() {
		t.Error("stdio should not be an HTTP transport")
	}
	if !TransportSSE.IsHTTP() {
		t.Error("sse should be an HTTP transport")
	}
	if !TransportStreamableHTTP.IsHTTP() {
		t.Error("streamablehttp should be an HTTP transport")
	}
}

func TestNewConfigDefaultsToStreamableHTTP(t *testing.T) {
	c := mustConfig(t, nil)

	if c.ParsedTransport != TransportStreamableHTTP {
		t.Errorf("default transport = %q, want %q", c.ParsedTransport, TransportStreamableHTTP)
	}
	if c.HTTPPort != 8080 {
		t.Errorf("default HTTPPort = %d, want 8080", c.HTTPPort)
	}
	if c.HTTPPath != "/mcp" {
		t.Errorf("default HTTPPath = %q, want /mcp", c.HTTPPath)
	}
	if c.AuthToken != "" {
		t.Errorf("default AuthToken = %q, want empty (auth disabled)", c.AuthToken)
	}
}

// The legacy SSE_PORT must keep working for deployments that predate HTTP_PORT.
func TestNewConfigLegacySSEPort(t *testing.T) {
	t.Run("SSE_PORT alone is honored", func(t *testing.T) {
		c := mustConfig(t, map[string]string{"SSE_PORT": "9090"})
		if c.HTTPPort != 9090 {
			t.Errorf("HTTPPort = %d, want 9090", c.HTTPPort)
		}
	})

	t.Run("HTTP_PORT wins over SSE_PORT", func(t *testing.T) {
		c := mustConfig(t, map[string]string{"SSE_PORT": "9090", "HTTP_PORT": "7070"})
		if c.HTTPPort != 7070 {
			t.Errorf("HTTPPort = %d, want 7070", c.HTTPPort)
		}
	})
}

func TestNewConfigRejectsInvalid(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"unknown transport", map[string]string{"TRANSPORT": "carrier-pigeon"}},
		{"path without leading slash", map[string]string{"HTTP_PATH": "mcp"}},
		{"port out of range", map[string]string{"HTTP_PORT": "70000"}},
		{"metrics port collision", map[string]string{
			"METRICS_ENABLED": "true", "METRICS_PORT": "8080", "HTTP_PORT": "8080",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if _, err := NewConfig(); err == nil {
				t.Errorf("NewConfig() with %v = nil error, want error", tt.env)
			}
		})
	}
}

// A bad HTTP_PATH is irrelevant when the transport never opens a socket.
func TestNewConfigSkipsHTTPValidationForStdio(t *testing.T) {
	c := mustConfig(t, map[string]string{"TRANSPORT": "stdio", "HTTP_PATH": "not-a-path"})
	if c.ParsedTransport != TransportStdio {
		t.Errorf("transport = %q, want stdio", c.ParsedTransport)
	}
}

func mustConfig(t *testing.T, environ map[string]string) *Config {
	t.Helper()

	clearEnv(t)

	for k, v := range environ {
		t.Setenv(k, v)
	}

	c, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() returned unexpected error: %v", err)
	}

	return c
}

// clearEnv unsets every variable NewConfig reads so a developer's ambient
// shell (or a .env already exported) cannot change what these tests assert.
func clearEnv(t *testing.T) {
	t.Helper()

	vars := []string{
		"LOG_LEVEL", "METRICS_ENABLED", "METRICS_PORT", "SERVER_NAME",
		"WSDOT_API_KEY", "WSDOT_API_TIMEOUT", "LOCAL", "TRANSPORT",
		"HTTP_PORT", "HTTP_PATH", "SSE_PORT", "MCP_AUTH_TOKEN", "MCP_STATELESS",
		"MCP_HEARTBEAT_INTERVAL", "SHUTDOWN_TIMEOUT", "TRACING_ENABLED",
		"TRACING_SAMPLERATE", "TRACING_SERVICE", "TRACING_VERSION",
	}

	for _, v := range vars {
		// t.Setenv registers the restore; Unsetenv then gives us a clean slate
		// that is still rolled back when the test finishes.
		t.Setenv(v, "")

		if err := os.Unsetenv(v); err != nil {
			t.Fatalf("could not unset %s: %v", v, err)
		}
	}
}
