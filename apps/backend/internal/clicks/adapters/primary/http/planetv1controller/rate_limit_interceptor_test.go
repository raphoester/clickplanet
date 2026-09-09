package planetv1controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ctxutil"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
	"github.com/stretchr/testify/require"
)

type fakeLimiter struct {
	allow bool
	keys  []string
}

func (l *fakeLimiter) Allow(key string) bool {
	l.keys = append(l.keys, key)
	return l.allow
}

// fakeRequest names a procedure, which is all the interceptor reads. The
// server fills the spec in for real; connect.NewRequest builds a client-side
// request whose spec is empty and read-only.
type fakeRequest struct {
	connect.AnyRequest
	spec connect.Spec
}

func (r fakeRequest) Spec() connect.Spec {
	return r.spec
}

func rateLimit(ctx context.Context, limiter ClickLimiter, procedure string) (bool, error) {
	handlerRan := false
	next := connect.UnaryFunc(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		handlerRan = true
		return connect.NewResponse(&planetv1.ClickResponse{}), nil
	})

	req := fakeRequest{spec: connect.Spec{Procedure: procedure}}

	_, err := NewRateLimitInterceptor(limiter).WrapUnary(next)(ctx, req)

	return handlerRan, err
}

func TestRateLimitInterceptor(t *testing.T) {
	t.Run("lets an allowed click through", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}

		ran, err := rateLimit(context.Background(), limiter, planetv1connect.ClickServiceClickProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("refuses a click over the limit without reaching the handler", func(t *testing.T) {
		limiter := &fakeLimiter{allow: false}

		ran, err := rateLimit(context.Background(), limiter, planetv1connect.ClickServiceClickProcedure)

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		require.False(t, ran, "a refused click must not reach the domain")
	})

	t.Run("keys on the source IP from the context", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := ctxutil.AddIPToContext(context.Background(), "1.2.3.4")

		_, err := rateLimit(ctx, limiter, planetv1connect.ClickServiceClickProcedure)

		require.NoError(t, err)
		require.Equal(t, []string{"1.2.3.4"}, limiter.keys)
	})

	t.Run("leaves the read procedures alone", func(t *testing.T) {
		limiter := &fakeLimiter{allow: false}

		for _, procedure := range []string{
			planetv1connect.ClickServiceGetMapProcedure,
			planetv1connect.ClickServiceMapDensityProcedure,
		} {
			ran, err := rateLimit(context.Background(), limiter, procedure)

			require.NoError(t, err)
			require.True(t, ran)
		}

		require.Empty(t, limiter.keys, "reads are never even offered to the limiter")
	})
}

// TestRateLimitOverHTTP walks the whole chain a real click goes through — the
// IP middleware, both interceptors and the generated handler — because that is
// where the parts can disagree: an interceptor order that lets the error
// mapping rewrite the refusal, or a limiter keyed on an IP the middleware
// never put on the context.
func TestRateLimitOverHTTP(t *testing.T) {
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	limiter := ratelimit.New(ratelimit.Config{PerSecond: 1, Burst: 10}, clock)

	server := clickServer(t, connect.WithInterceptors(
		NewErrorInterceptor(nil),
		NewRateLimitInterceptor(limiter),
	))

	click := func(ip string) error {
		req := connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"})
		req.Header().Set("X-Real-IP", ip)
		_, err := planetv1connect.NewClickServiceClient(server.Client(), server.URL).
			Click(context.Background(), req)
		return err
	}

	for i := 0; i < 10; i++ {
		require.NoErrorf(t, click("1.2.3.4"), "click %d should be allowed", i)
	}

	err := click("1.2.3.4")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	// The same refusal seen the way a browser sees it, since the status is
	// what a proxy or a client-side backoff keys on.
	require.Equal(t, http.StatusTooManyRequests, clickStatus(t, server, "1.2.3.4"))

	require.NoError(t, click("5.6.7.8"), "another address has its own allowance")

	clock.advance(time.Second)
	require.NoError(t, click("1.2.3.4"), "a second later the bucket has a token again")
}

// clickStatus posts a Connect unary request by hand and reports the HTTP
// status, which the generated client hides behind a code.
func clickStatus(t *testing.T, server *httptest.Server, ip string) int {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost,
		server.URL+planetv1connect.ClickServiceClickProcedure,
		strings.NewReader(`{"tileId":1,"countryId":"fr"}`))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	req.Header.Set("X-Real-IP", ip)

	res, err := server.Client().Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	return res.StatusCode
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.now = c.now.Add(d)
}
