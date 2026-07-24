package mcpserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/michaelpeterswa/mcp-wsdot/internal/config"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	// LivenessPath and ReadinessPath are unauthenticated so Kubernetes probes
	// work without wiring the bearer token into the pod spec.
	LivenessPath  = "/healthz"
	ReadinessPath = "/readyz"

	// readHeaderTimeout bounds the slowloris window. The body and response
	// deadlines are deliberately left unset: streamable HTTP and SSE both hold
	// responses open indefinitely, so a WriteTimeout would sever live streams.
	readHeaderTimeout = 10 * time.Second
)

// newServeMux mounts the MCP handler at mcpPath alongside the health endpoints.
// Only the MCP handler is authenticated and traced; probes must stay cheap and
// reachable.
func newServeMux(c *config.Config, mcpPath string, mcpHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc(LivenessPath, healthHandler)
	mux.HandleFunc(ReadinessPath, healthHandler)

	handler := otelhttp.NewHandler(mcpHandler, "mcp")
	if c.AuthToken != "" {
		handler = requireBearerToken(c.AuthToken, handler)
	}
	mux.Handle(mcpPath, handler)

	return mux
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

// requireBearerToken rejects requests that do not carry the configured token in
// an "Authorization: Bearer <token>" header.
func requireBearerToken(token string, next http.Handler) http.Handler {
	want := []byte(token)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		// The comparison is constant time, but its result still leaks whether
		// the lengths matched; that is acceptable for a shared static token.
		if !ok || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			slog.Warn("rejected unauthenticated mcp request",
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("path", r.URL.Path),
			)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// serve runs handler until ctx is cancelled, then drains in-flight requests
// within c.ShutdownTimeout. transportShutdown closes any MCP sessions the
// transport is tracking and is called before the listener drains.
func serve(
	ctx context.Context,
	c *config.Config,
	handler http.Handler,
	transportShutdown func(context.Context) error,
) error {
	addr := fmt.Sprintf(":%d", c.HTTPPort)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serveErr := make(chan error, 1)

	go func() {
		slog.Info("mcp server listening",
			slog.String("addr", addr),
			slog.String("transport", string(c.ParsedTransport)),
			slog.Bool("authenticated", c.AuthToken != ""),
			slog.Bool("stateless", c.Stateless),
		)

		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}

		serveErr <- err
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down mcp server", slog.Duration("timeout", c.ShutdownTimeout))

	// Deliberately built from context.Background(): ctx is already cancelled,
	// so deriving from it would expire the drain immediately.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), c.ShutdownTimeout)
	defer cancel()

	var errs []error

	if transportShutdown != nil {
		if err := transportShutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("could not shut down mcp transport: %w", err))
		}
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		errs = append(errs, fmt.Errorf("could not shut down http server: %w", err))
	}

	if err := <-serveErr; err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
