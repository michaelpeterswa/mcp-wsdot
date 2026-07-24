package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/michaelpeterswa/mcp-wsdot/internal/config"
)

// okHandler stands in for the MCP handler and records whether it was reached.
func okHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantPassed bool
	}{
		{"correct token", "Bearer s3cret", http.StatusOK, true},
		{"missing header", "", http.StatusUnauthorized, false},
		{"wrong token", "Bearer nope", http.StatusUnauthorized, false},
		{"token as prefix of secret", "Bearer s3c", http.StatusUnauthorized, false},
		{"secret as prefix of token", "Bearer s3cretextra", http.StatusUnauthorized, false},
		{"bare token without scheme", "s3cret", http.StatusUnauthorized, false},
		{"wrong scheme", "Basic s3cret", http.StatusUnauthorized, false},
		{"scheme is case sensitive", "bearer s3cret", http.StatusUnauthorized, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reached bool
			h := requireBearerToken("s3cret", okHandler(&reached))

			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if reached != tt.wantPassed {
				t.Errorf("handler reached = %v, want %v", reached, tt.wantPassed)
			}
			if tt.wantStatus == http.StatusUnauthorized {
				if got := rec.Header().Get("WWW-Authenticate"); got == "" {
					t.Error("401 response is missing a WWW-Authenticate header")
				}
			}
		})
	}
}

// Probes must stay reachable without credentials, otherwise every pod fails its
// readiness check the moment a token is configured.
func TestHealthEndpointsBypassAuth(t *testing.T) {
	var reached bool
	c := &config.Config{AuthToken: "s3cret", HTTPPath: "/mcp"}
	mux := newServeMux(c, c.HTTPPath, okHandler(&reached))

	for _, path := range []string{LivenessPath, ReadinessPath} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", rec.Code)
			}
		})
	}

	if reached {
		t.Error("a health probe was routed to the MCP handler")
	}
}

func TestServeMuxRoutesMCPPath(t *testing.T) {
	t.Run("auth enforced when token set", func(t *testing.T) {
		var reached bool
		c := &config.Config{AuthToken: "s3cret", HTTPPath: "/mcp"}
		mux := newServeMux(c, c.HTTPPath, okHandler(&reached))

		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
		if reached {
			t.Error("unauthenticated request reached the MCP handler")
		}
	})

	t.Run("no auth when token empty", func(t *testing.T) {
		var reached bool
		c := &config.Config{HTTPPath: "/mcp"}
		mux := newServeMux(c, c.HTTPPath, okHandler(&reached))

		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
		if !reached {
			t.Error("request did not reach the MCP handler")
		}
	})

	// The SSE transport mounts at "/" because SSEServer routes its own
	// /sse and /message paths internally.
	t.Run("catch-all mount still yields health paths", func(t *testing.T) {
		var reached bool
		c := &config.Config{HTTPPath: "/mcp"}
		mux := newServeMux(c, "/", okHandler(&reached))

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, LivenessPath, nil))
		if rec.Code != http.StatusOK || reached {
			t.Errorf("liveness probe fell through to the MCP handler (code=%d reached=%v)", rec.Code, reached)
		}

		rec = httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/sse", nil))
		if !reached {
			t.Error("/sse did not reach the MCP handler")
		}
	})
}
