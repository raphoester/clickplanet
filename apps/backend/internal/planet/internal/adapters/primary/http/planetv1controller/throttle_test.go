package planetv1controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/stretchr/testify/require"
)

// throttledServer wires the throttle where it now lives — inside the click
// chain — and serves it over HTTP, which is the only way to see that a refusal
// still reaches a browser as a 429 now that no interceptor produces one.
func throttledServer(t *testing.T, config cpratelimit.Config) (*httptest.Server, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	limiter := cpratelimit.New(config, clock)

	server := clickServerWith(t, throttle_click.New(stubService{}, limiter), limiter,
		connect.WithInterceptors(errorNet()))

	return server, clock
}

func clickAs(t *testing.T, server *httptest.Server, ip string) (*connect.Response[planetv1.ClickResponse], error) {
	t.Helper()

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"})
	req.Header().Set("X-Real-IP", ip)

	return planetv1connect.NewClickServiceClient(server.Client(), server.URL).Click(t.Context(), req)
}

func TestThrottleOverHTTP(t *testing.T) {
	server, clock := throttledServer(t, cpratelimit.Config{PerSecond: 1, Burst: 10})

	for i := range 10 {
		_, err := clickAs(t, server, "1.2.3.4")
		require.NoErrorf(t, err, "click %d should be allowed", i)
	}

	_, err := clickAs(t, server, "1.2.3.4")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
	require.Equal(t, http.StatusTooManyRequests, clickStatus(t, server, "1.2.3.4"))

	_, err = clickAs(t, server, "5.6.7.8")
	require.NoError(t, err, "another address has its own allowance")

	clock.Advance(time.Second)
	_, err = clickAs(t, server, "1.2.3.4")
	require.NoError(t, err, "a second later the bucket has a token again")
}

func TestTheBudgetRidesOnEveryAnswer(t *testing.T) {
	t.Run("an accepted click says what is left, and the policy to replay it", func(t *testing.T) {
		server, _ := throttledServer(t, cpratelimit.Config{PerSecond: 2, Burst: 5})

		res, err := clickAs(t, server, "1.2.3.4")
		require.NoError(t, err)

		budget := res.Msg.GetBudget()
		require.InDelta(t, float64(4), budget.GetTokens(), 1e-9, "the click just spent one of five")
		require.Equal(t, uint32(5), budget.GetCapacity())
		require.InDelta(t, float64(2), budget.GetRefillPerSecond(), 1e-9)
	})

	t.Run("a refused click carries the wait on the error", func(t *testing.T) {
		server, clock := throttledServer(t, cpratelimit.Config{PerSecond: 2, Burst: 5})

		for i := range 5 {
			_, err := clickAs(t, server, "1.2.3.4")
			require.NoErrorf(t, err, "click %d should be allowed", i)
		}

		clock.Advance(200 * time.Millisecond)

		_, err := clickAs(t, server, "1.2.3.4")
		require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

		budget := budgetDetail(t, err)
		require.InDelta(t, 0.4, budget.GetTokens(), 1e-6, "the next click is 300ms away")
		require.Equal(t, uint32(5), budget.GetCapacity())
	})

	t.Run("GetBudget reads the allowance without spending it", func(t *testing.T) {
		server, _ := throttledServer(t, cpratelimit.Config{PerSecond: 2, Burst: 5})

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

		_, err := clickAs(t, server, "1.2.3.4")
		require.NoError(t, err)

		for range 3 {
			require.InDelta(t, float64(4), read().GetTokens(), 1e-9, "reading is free")
		}
	})
}

func TestTheBudgetIsAbsentWithoutAThrottle(t *testing.T) {
	server := clickServer(t, connect.WithInterceptors(errorNet()))

	res, err := clickAs(t, server, "1.2.3.4")
	require.NoError(t, err)

	require.Nil(t, res.Msg.GetBudget(), "a server that does not throttle promises no allowance")
}
