package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"alpineworks.io/ootel"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/michaelpeterswa/go-mcp-template/internal/config"
	"github.com/michaelpeterswa/go-mcp-template/internal/logging"
	"github.com/michaelpeterswa/go-mcp-template/internal/mcpserver"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "error"
	}

	slogLevel, err := logging.LogLevelToSlogLevel(logLevel)
	if err != nil {
		log.Fatalf("could not convert log level: %s", err)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})))
	c, err := config.NewConfig()
	if err != nil {
		slog.Error("could not create config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx := context.Background()

	exporterType := ootel.ExporterTypePrometheus
	if c.Local {
		exporterType = ootel.ExporterTypeOTLPGRPC
	}

	ootelClient := ootel.NewOotelClient(
		ootel.WithMetricConfig(
			ootel.NewMetricConfig(
				c.MetricsEnabled,
				exporterType,
				c.MetricsPort,
			),
		),
		ootel.WithTraceConfig(
			ootel.NewTraceConfig(
				c.TracingEnabled,
				c.TracingSampleRate,
				c.TracingService,
				c.TracingVersion,
			),
		),
	)

	shutdown, err := ootelClient.Init(ctx)
	if err != nil {
		slog.Error("could not create ootel client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(5 * time.Second))
	if err != nil {
		slog.Error("could not create runtime metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}

	err = host.Start()
	if err != nil {
		slog.Error("could not create host metrics", slog.String("error", err.Error()))
		os.Exit(1)
	}

	defer func() {
		_ = shutdown(ctx)
	}()

	tools := []mcpserver.Tool{
		mcpserver.NewTool(
			mcp.NewTool(
				"add numbers",
				mcp.WithDescription("add two numbers"),
				mcp.WithNumber("a1",
					mcp.Required(),
					mcp.Description("the first number"),
				),
				mcp.WithNumber("a2",
					mcp.Required(),
					mcp.Description("the second number"),
				),
			),
			addHandler,
		),
	}

	err = mcpserver.StartServer(ctx, c, mcpserver.WithTools(tools))
	if err != nil {
		slog.Error("could not start server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

type Result struct {
	Result float64 `json:"result"`
}

func addHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	a1, ok := request.Params.Arguments["a1"].(float64)
	if !ok {
		return nil, errors.New("name must be a string")
	}

	a2, ok := request.Params.Arguments["a2"].(float64)
	if !ok {
		return nil, errors.New("name must be a string")
	}

	result := Result{
		Result: a1 + a2,
	}

	resultJson, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("could not marshal result: %w", err)
	}

	return mcp.NewToolResultText(string(resultJson)), nil
}
