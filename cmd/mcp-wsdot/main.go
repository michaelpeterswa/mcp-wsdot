package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// get_current_time resolves America/Los_Angeles at runtime. Embedding the
	// tz database keeps that working on distroless and scratch images, which
	// ship no /usr/share/zoneinfo.
	_ "time/tzdata"

	"alpineworks.io/ootel"
	"alpineworks.io/wsdot"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

	// Logs go to stderr, not stdout: the stdio transport speaks JSON-RPC over
	// stdout and any log line written there corrupts the stream. Docker and
	// Kubernetes collect both streams identically.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slogLevel,
	})))
	c, err := config.NewConfig()
	if err != nil {
		slog.Error("could not create config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Kubernetes sends SIGTERM and then waits out terminationGracePeriodSeconds
	// before SIGKILL; cancelling ctx here is what lets in-flight tool calls
	// finish during a rolling update.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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

	// Every tool here only reads upstream data, so annotate them read-only and
	// non-destructive. get_current_time is a local clock (closed world); the
	// WSDOT-backed tools reach an external API (open world). Clients such as
	// openclaw use these hints when deciding what to auto-approve.
	//
	// Descriptions deliberately include the words people actually use ("ferry",
	// "boat", "sailing", "Washington State Ferries", terminal/route names) so
	// the tools are discoverable across a wide range of phrasings, and they
	// spell out the call chain (routes first, then schedules by routeID).
	tools := []mcpserver.Tool{
		mcpserver.NewTool(
			mcp.NewTool(
				"get_route_schedules",
				mcp.WithTitleAnnotation("List ferry routes"),
				mcp.WithDescription(
					"List every Washington State Ferries (WSF/WSDOT) route with its "+
						"name and numeric RouteID — for example Seattle-Bainbridge, "+
						"Seattle-Bremerton, Edmonds-Kingston, Mukilteo-Clinton, and the "+
						"Anacortes / San Juan Islands routes. Call this first to find the "+
						"routeID that get_schedules_today_by_route_id needs. Use it for "+
						"questions like \"which ferry routes exist\" or to map a pair of "+
						"terminal names (e.g. Seattle to Bremerton) to a route.",
				),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithOpenWorldHintAnnotation(true),
			),
			whc.GetRouteSchedulesHandler,
		),
		mcpserver.NewTool(
			mcp.NewTool(
				"get_schedules_today_by_route_id",
				mcp.WithTitleAnnotation("Today's ferry sailing times for a route"),
				mcp.WithDescription(
					"Get today's Washington State Ferries (WSF) sailing times (ferry "+
						"departures) for one route, given its numeric routeID. Answers "+
						"questions like \"when is the next boat\", \"the next few sailings\", "+
						"or \"the last ferry tonight\" for a route. The routeID comes from "+
						"get_route_schedules — call that first if you only know the "+
						"terminal or route names. Set onlyRemainingTime to return just the "+
						"sailings still to come today.",
				),
				mcp.WithNumber("routeID",
					mcp.Description("Numeric RouteID from get_route_schedules (the RouteID field)."),
					mcp.Required(),
				),
				mcp.WithBoolean("onlyRemainingTime",
					mcp.Description("If true, return only sailings still remaining today rather than the full day's schedule."),
				),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithOpenWorldHintAnnotation(true),
			),
			whc.GetSchedulesTodayByRouteIDHandler,
		),
		mcpserver.NewTool(
			mcp.NewTool(
				"get_current_time",
				mcp.WithTitleAnnotation("Current Pacific time"),
				mcp.WithDescription(
					"Get the current date and time in the US Pacific timezone "+
						"(America/Los_Angeles, PST/PDT) — the timezone all ferry "+
						"schedules are in. Use it to reason about \"now\", \"today\", "+
						"\"the next sailing\", or \"tonight\" relative to schedule times.",
				),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithOpenWorldHintAnnotation(false),
			),
			handlers.CurrentTimeHandler,
		),
	}

	// Server-level instructions tell client/agent routers what this whole
	// server is for, so it surfaces for ferry/transit prompts.
	instructions := "Washington State Ferries (WSF/WSDOT) schedule server. " +
		"Provides ferry routes and today's sailing (departure) times for the " +
		"Washington State Ferries system — Seattle-Bainbridge, Seattle-Bremerton, " +
		"Edmonds-Kingston, Mukilteo-Clinton, Fauntleroy-Vashon-Southworth, the " +
		"Anacortes / San Juan Islands routes, and more. Typical flow: call " +
		"get_route_schedules to resolve a route or terminal-name pair to a " +
		"routeID, then get_schedules_today_by_route_id for that route's times. " +
		"get_current_time gives the Pacific time schedules are expressed in."

	err = mcpserver.StartServer(ctx, c,
		mcpserver.WithServerOptions(server.WithInstructions(instructions)),
		mcpserver.WithTools(tools),
	)
	if err != nil {
		slog.Error("could not start server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
