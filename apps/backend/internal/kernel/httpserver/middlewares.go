package httpserver

import (
	"net"
	"net/http"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
)

func MiddlewareStack(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// IPReaderMiddleware puts the caller's address on the context, where the
// rate limiter and the click metrics read it from.
//
// X-Real-IP comes first because in production the only address the socket
// knows is Cloudflare's or Caddy's; the reverse proxy is expected to set that
// header from the real client and to overwrite whatever the client sent, since
// nothing here can tell a forged one from a genuine one. Without a proxy — a
// local run, a direct container — the socket is the truth and there is no
// header, which is why the fallback is the peer address rather than a shared
// "unknown": one bucket for every anonymous caller would rate limit the whole
// game as if it were a single player.
func IPReaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Real-IP")
		if ip == "" {
			ip = remoteIP(r)
		}

		ctx := ctxutil.AddIPToContext(r.Context(), ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// remoteIP is the peer address with its port stripped, so that two
// connections from the same machine land in the same bucket.
func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func NewLoggingMiddleware(logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapped := &wrappedWriter{ResponseWriter: w}
			next.ServeHTTP(wrapped, r)
			logger.Info(
				"new request on web server",
				lf.String("method", r.Method),
				lf.String("uri", r.RequestURI),
				lf.Int("status_code", wrapped.code),
			)
		})
	}
}

type wrappedWriter struct {
	code int
	http.ResponseWriter
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
