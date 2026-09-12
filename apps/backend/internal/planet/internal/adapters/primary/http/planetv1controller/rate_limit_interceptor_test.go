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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/stretchr/testify/require"
)

type fakeLimiter struct {
	allow bool
	state cpratelimit.State
	keys  []string
}

func (l *fakeLimiter) Take(key string) (bool, cpratelimit.State) {
	l.keys = append(l.keys, key)
	return l.allow, l.state
}

type fakeRequest struct {
	connect.AnyRequest
	spec   connect.Spec
	header http.Header
}

func (r fakeRequest) Spec() connect.Spec {
	return r.spec
}

func (r fakeRequest) Header() http.Header {
	if r.header == nil {
		return http.Header{}
	}
	return r.header
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

		ran, err := rateLimit(t.Context(), limiter, planetv1connect.ClickServiceClickProcedure)

		require.NoError(t, err)
		require.True(t, ran)
	})

	t.Run("refuses a click over the limit without reaching the handler", func(t *testing.T) {
		limiter := &fakeLimiter{allow: false}

		ran, err := rateLimit(t.Context(), limiter, planetv1connect.ClickServiceClickProcedure)

		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
		require.False(t, ran, "a refused click must not reach the domain")
	})

	t.Run("keys on the source IP from the context", func(t *testing.T) {
		limiter := &fakeLimiter{allow: true}
		ctx := cpctx.AddIPToContext(t.Context(), "1.2.3.4")

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
			ran, err := rateLimit(t.Context(), limiter, procedure)

			require.NoError(t, err)
			require.True(t, ran)
		}

		require.Empty(t, limiter.keys, "reads are never even offered to the limiter")
	})
}

func TestRateLimitOverHTTP(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	limiter := cpratelimit.New(cpratelimit.Config{PerSecond: 1, Burst: 10}, clock)

	server := clickServer(t, connect.WithInterceptors(
		NewErrorInterceptor(nil),
		NewRateLimitInterceptor(limiter),
	))

	click := func(ip string) error {
		req := connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"})
		req.Header().Set("X-Real-IP", ip)
		_, err := planetv1connect.NewClickServiceClient(server.Client(), server.URL).
			Click(t.Context(), req)
		return err
	}

	for i := 0; i < 10; i++ {
		require.NoErrorf(t, click("1.2.3.4"), "click %d should be allowed", i)
	}

	err := click("1.2.3.4")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	require.Equal(t, http.StatusTooManyRequests, clickStatus(t, server, "1.2.3.4"))

	require.NoError(t, click("5.6.7.8"), "another address has its own allowance")

	clock.Advance(time.Second)
	require.NoError(t, click("1.2.3.4"), "a second later the bucket has a token again")
}

func clickStatus(t *testing.T, server *httptest.Server, ip string) int {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
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

func TestTheBudgetRidesOnEveryAnswer(t *testing.T) {
	newServer := func(t *testing.T) (*httptest.Server, *cpratelimit.Limiter, *cptime.FixedClock) {
		t.Helper()

		clock := cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
		limiter := cpratelimit.New(cpratelimit.Config{PerSecond: 2, Burst: 5}, clock)

		return clickServerReading(t, limiter, connect.WithInterceptors(
			NewErrorInterceptor(nil),
			NewRateLimitInterceptor(limiter),
		)), limiter, clock
	}

	click := func(t *testing.T, server *httptest.Server) (*connect.Response[planetv1.ClickResponse], error) {
		t.Helper()

		req := connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"})
		req.Header().Set("X-Real-IP", "1.2.3.4")

		return planetv1connect.NewClickServiceClient(server.Client(), server.URL).
			Click(t.Context(), req)
	}

	t.Run("an accepted click says what is left, and the policy to replay it", func(t *testing.T) {
		server, _, _ := newServer(t)

		res, err := click(t, server)
		require.NoError(t, err)

		budget := res.Msg.GetBudget()
		require.InDelta(t, float64(4), budget.GetTokens(), 1e-9, "the click just spent one of five")
		require.Equal(t, uint32(5), budget.GetCapacity())
		require.InDelta(t, float64(2), budget.GetRefillPerSecond(), 1e-9)
	})

	t.Run("a refused click carries the wait on the error", func(t *testing.T) {
		server, _, clock := newServer(t)

		for i := 0; i < 5; i++ {
			_, err := click(t, server)
			require.NoErrorf(t, err, "click %d should be allowed", i)
		}

		clock.Advance(200 * time.Millisecond)

		_, err := click(t, server)
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

		budget := budgetDetail(t, err)
		require.InDelta(t, 0.4, budget.GetTokens(), 1e-6, "the next click is 300ms away")
		require.Equal(t, uint32(5), budget.GetCapacity())
	})

	t.Run("GetBudget reads the allowance without spending it", func(t *testing.T) {
		server, _, _ := newServer(t)

		read := func() *planetv1.ClickBudget {
			t.Helper()

			req := connect.NewRequest(&planetv1.GetBudgetRequest{})
			req.Header().Set("X-Real-IP", "1.2.3.4")

			res, err := planetv1connect.NewClickServiceClient(server.Client(), server.URL).
				GetBudget(t.Context(), req)
			require.NoError(t, err)

			return res.Msg.GetBudget()
		}

		require.InDelta(t, float64(5), read().GetTokens(), 1e-9, "an address that never clicked is full")

		_, err := click(t, server)
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			require.InDelta(t, float64(4), read().GetTokens(), 1e-9, "reading is free")
		}
	})
}

func TestTheBudgetIsAbsentWithoutALimiter(t *testing.T) {
	server := clickServer(t, connect.WithInterceptors(NewErrorInterceptor(nil)))

	res, err := planetv1connect.NewClickServiceClient(server.Client(), server.URL).
		Click(t.Context(), connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"}))
	require.NoError(t, err)

	require.Nil(t, res.Msg.GetBudget(), "a server that does not throttle promises no allowance")
}

func budgetDetail(t *testing.T, err error) *planetv1.ClickBudget {
	t.Helper()

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)

	for _, detail := range connectErr.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			continue
		}

		if budget, ok := value.(*planetv1.ClickBudget); ok {
			return budget
		}
	}

	t.Fatal("the refusal carried no budget")
	return nil
}
