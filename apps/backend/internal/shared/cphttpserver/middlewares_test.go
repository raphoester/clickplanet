package cphttpserver_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddlewareKeepsTheWriterFlushable(t *testing.T) {
	var flushable bool

	middleware := cphttpserver.NewLoggingMiddleware(slog.New(slog.DiscardHandler))
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, flushable = w.(http.Flusher)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))

	require.True(t, flushable)
}

func TestIPReaderMiddleware(t *testing.T) {
	readIP := func(r *http.Request) string {
		seen := ""
		handler := cphttpserver.IPReaderMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = cpctx.GetSourceIP(r.Context())
		}))
		handler.ServeHTTP(httptest.NewRecorder(), r)
		return seen
	}

	t.Run("prefers the address the reverse proxy resolved", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/planet.v1.ClickService/Click", nil)
		r.RemoteAddr = "10.0.0.1:4242"
		r.Header.Set("X-Real-IP", "1.2.3.4")

		require.Equal(t, "1.2.3.4", readIP(r))
	})

	t.Run("falls back to the peer address, without its port", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/planet.v1.ClickService/Click", nil)
		r.RemoteAddr = "192.168.1.7:51234"

		require.Equal(t, "192.168.1.7", readIP(r))
	})

	t.Run("keeps a peer address that carries no port", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/planet.v1.ClickService/Click", nil)
		r.RemoteAddr = "@"

		require.Equal(t, "@", readIP(r))
	})
}

// A custom header makes a cross-origin POST preflighted, and a preflight that
// does not list it fails the request before the handler ever sees it — so
// omitting this would refuse every click from the deployed frontend.
func TestCorsMiddlewareAllowsTheSessionHeader(t *testing.T) {
	handler := cphttpserver.CorsMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/planet.v1.ClickService/Click", nil))

	allowed := recorder.Header().Get("Access-Control-Allow-Headers")
	require.Contains(t, allowed, cpconnect.SessionHeader)
	require.Contains(t, allowed, "Content-Type")
	require.Contains(t, allowed, "Connect-Protocol-Version")
}
