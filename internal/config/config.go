package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

var (
	AppVersion = "unset"
)

type Config struct {
	LogLevel string `env:"LOG_LEVEL" envDefault:"error"`

	MetricsEnabled bool `env:"METRICS_ENABLED" envDefault:"false"`
	MetricsPort    int  `env:"METRICS_PORT" envDefault:"8081"`

	ServerName string `env:"SERVER_NAME" envDefault:"go-mcp-template"`

	Local     bool   `env:"LOCAL" envDefault:"false"`
	Transport string `env:"TRANSPORT" envDefault:"sse"`

	SSEPort int `env:"SSE_PORT" envDefault:"8080"`

	TracingEnabled    bool    `env:"TRACING_ENABLED" envDefault:"false"`
	TracingSampleRate float64 `env:"TRACING_SAMPLERATE" envDefault:"0.01"`
	TracingService    string  `env:"TRACING_SERVICE" envDefault:"go-mcp-template"`
	TracingVersion    string  `env:"TRACING_VERSION"`
}

func NewConfig() (*Config, error) {
	var cfg Config

	err := env.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
