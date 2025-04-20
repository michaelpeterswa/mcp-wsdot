package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"alpineworks.io/ootel"
	"alpineworks.io/wsdot"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/michaelpeterswa/mcp-wsdot/internal/config"
	"github.com/michaelpeterswa/mcp-wsdot/internal/handlers"
	"github.com/michaelpeterswa/mcp-wsdot/internal/logging"
	"github.com/michaelpeterswa/mcp-wsdot/internal/mcpserver"
	"go.opentelemetry.io/contrib/instrumentation/host"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	httpClient := http.Client{
		Timeout:   c.WSDOTAPITimeout,
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	wsdotClient, err := wsdot.NewWSDOTClient(wsdot.WithAPIKey(c.WSDOTAPIKey), wsdot.WithHTTPClient(&httpClient))
	if err != nil {
		slog.Error("could not create wsdot client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	whc, err := handlers.NewWSDOTHandlerClient(wsdotClient)
	if err != nil {
		slog.Error("could not create wsdot handler client", slog.String("error", err.Error()))
		os.Exit(1)
	}

	tools := []mcpserver.Tool{
		mcpserver.NewTool(
			mcp.NewTool(
				"get_route_schedules",
				mcp.WithDescription("get the route names and ids for a schedule"),
			),
			whc.GetRouteSchedulesHandler,
		),
		mcpserver.NewTool(
			mcp.NewTool(
				"get_schedules_today_by_route_id",
				mcp.WithDescription("get the schedule for a route today by route id"),
				mcp.WithNumber("routeID",
					mcp.Description("the route id"),
					mcp.Required(),
				),
				mcp.WithBoolean("onlyRemainingTime",
					mcp.Description("only return the remaining sailing times"),
				),
			),
			whc.GetSchedulesTodayByRouteIDHandler,
		),
	}

	err = mcpserver.StartServer(ctx, c, mcpserver.WithTools(tools))
	if err != nil {
		slog.Error("could not start server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
