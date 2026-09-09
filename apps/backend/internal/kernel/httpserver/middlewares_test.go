package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/httpserver"
	"github.com/stretchr/testify/require"
)

func TestIPReaderMiddleware(t *testing.T) {
	readIP := func(r *http.Request) string {
		seen := ""
		handler := httpserver.IPReaderMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			seen = ctxutil.GetSourceIP(r.Context())
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
