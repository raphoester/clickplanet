package httpserver

import (
	"net"
	"net/http"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/connectutil"
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

// The session header has to be named here or the browser never sends it: a
// custom header makes a cross-origin POST preflighted, and a preflight that
// does not list it fails the actual request. deploy/vps/Caddyfile answers
// OPTIONS itself in production and carries the same list.
var allowedHeaders = strings.Join([]string{
	"Content-Type",
	"Connect-Protocol-Version",
	"Connect-Timeout-Ms",
	connectutil.SessionHeader,
}, ", ")

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
