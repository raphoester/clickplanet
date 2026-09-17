package planetv1controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_budget_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/click_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_budget_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpconnect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/stretchr/testify/require"
)

// throttledServer wires the throttle where it now lives — inside the click
// chain — and serves it over HTTP, which is the only way to see that a refusal
// still reaches a browser as a 429 now that no interceptor produces one.
func throttledServer(t *testing.T, config cpratelimit.Config) (*httptest.Server, *cptime.FixedClock) {
	t.Helper()
	return pricedServer(t, config, onePrice)
}

func pricedServer(t *testing.T, config cpratelimit.Config, pricer stubPricer) (*httptest.Server, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	limiter := cpratelimit.New("test", config, clock)

	server := clickServerWith(t, throttle_click.New(stubService{}, limiter, pricer, buckets), limiter,
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

func TestABigCountryPaysMorePerClick(t *testing.T) {
	server, _ := pricedServer(t, cpratelimit.Config{PerSecond: 1, Burst: 10},
		stubPricer{Cost: 1.5, Share: 0.4, NextShare: 0.7, NextCost: 2})

	res, err := clickAs(t, server, "1.2.3.4")
	require.NoError(t, err)

	budget := res.Msg.GetBudget()
	require.InDelta(t, 8.5/1.5, budget.GetTokens(), 1e-9, "8.5 tokens left are five and two thirds clicks")
	require.Equal(t, uint32(6), budget.GetCapacity(), "ten tokens hold six whole clicks at 1.5")
	require.InDelta(t, 1/1.5, budget.GetRefillPerSecond(), 1e-9)
	require.InDelta(t, 1.5, budget.GetCost(), 1e-9)
	require.InDelta(t, 0.4, budget.GetShare(), 1e-9)
	require.InDelta(t, 0.7, budget.GetNextShare(), 1e-9)
	require.InDelta(t, 2, budget.GetNextCost(), 1e-9)

	for i := range 5 {
		_, err := clickAs(t, server, "1.2.3.4")
		require.NoErrorf(t, err, "click %d should be allowed", i)
	}

	_, err = clickAs(t, server, "1.2.3.4")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "one token left does not pay for one and a half")
}

func TestTheBudgetIsAbsentWithoutAThrottle(t *testing.T) {
	server := clickServer(t, connect.WithInterceptors(errorNet()))

	res, err := clickAs(t, server, "1.2.3.4")
	require.NoError(t, err)

	require.Nil(t, res.Msg.GetBudget(), "a server that does not throttle promises no allowance")
}

// accountVerifier accepts a token that is an account id, and names that account. Prefixed with
// linkedPrefix, the account signed in with a provider.
type accountVerifier struct{}

const linkedPrefix = "linked "

func (accountVerifier) Verify(token string, _ string, _ time.Time) (*cpsession.Claims, error) {
	linked := strings.HasPrefix(token, linkedPrefix)
	account, err := uuid.Parse(strings.TrimPrefix(token, linkedPrefix))
	if err != nil {
		return nil, fmt.Errorf("not an account: %w", err)
	}
	return &cpsession.Claims{ID: "a-mint", Account: cpsession.AccountID(account), Linked: linked}, nil
}

func accountServer(t *testing.T, config clicks.ThrottleConfig) (*httptest.Server, *cpratelimit.Limiter, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	limiter := cpratelimit.New("test", config.Config, clock)

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewClickServiceHandler(
		ClickService{
			ClickHandler:     click_handler.New(throttle_click.New(stubService{}, limiter, onePrice, config.Buckets())),
			GetBudgetHandler: get_budget_handler.New(get_budget_usecase.New(limiter, onePrice, config.Buckets())),
		},
		connect.WithInterceptors(
			errorNet(),
			NewSessionInterceptor(accountVerifier{}, clock, true, prometheus.NewRegistry()),
			NewSessionReaderInterceptor(accountVerifier{}, clock),
		),
	))

	server := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return server, limiter, clock
}

func clickAsAccount(t *testing.T, server *httptest.Server, ip string, account uuid.UUID) error {
	t.Helper()

	_, err := clickWithToken(t, server, ip, account.String())
	return err
}

func clickWithToken(t *testing.T, server *httptest.Server, ip string, token string) (*connect.Response[planetv1.ClickResponse], error) {
	t.Helper()

	req := connect.NewRequest(&planetv1.ClickRequest{TileId: 1, CountryId: "fr"})
	req.Header().Set("X-Real-IP", ip)
	req.Header().Set(cpconnect.SessionHeader, token)

	return planetv1connect.NewClickServiceClient(server.Client(), server.URL).Click(t.Context(), req)
}

func accountNumber(i int) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte{byte(i)})
}

func TestManyAccountsOnOneScopeShareTheScopesBucket(t *testing.T) {
	server, _, _ := accountServer(t, clicks.ThrottleConfig{
		Config: cpratelimit.Config{PerSecond: 1, Burst: 10}, ScopeMultiplier: 3,
	})

	for i := range 3 {
		for click := range 10 {
			require.NoErrorf(t, clickAsAccount(t, server, "1.2.3.4", accountNumber(i)), "account %d click %d", i, click)
		}
	}

	err := clickAsAccount(t, server, "1.2.3.4", accountNumber(3))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "a fresh account behind a spent scope is refused")
	require.InDelta(t, 0.0, budgetDetail(t, err).GetTokens(), 1e-9, "and is told the scope's reading, not its own full bucket")
	require.Equal(t, uint32(30), budgetDetail(t, err).GetCapacity())

	require.NoError(t, clickAsAccount(t, server, "5.6.7.8", accountNumber(3)), "the same account elsewhere may click")
}

func TestOneAccountOnManyScopesSpendsOneAllowance(t *testing.T) {
	server, _, _ := accountServer(t, clicks.ThrottleConfig{Config: cpratelimit.Config{PerSecond: 1, Burst: 10}})

	for click := range 10 {
		require.NoErrorf(t, clickAsAccount(t, server, fmt.Sprintf("10.0.0.%d", click), accountNumber(0)), "click %d", click)
	}

	err := clickAsAccount(t, server, "10.0.0.99", accountNumber(0))
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "a new address is not a new allowance")
}

func TestABoostDoesNotWidenTheScopesBucket(t *testing.T) {
	server, limiter, clock := accountServer(t, clicks.ThrottleConfig{
		Config: cpratelimit.Config{PerSecond: 1, Burst: 10}, ScopeMultiplier: 2,
	})

	boosted := accountNumber(0)
	limiter.Boost("account:"+boosted.String(), 3, clock.Now().Add(time.Minute))
	for click := range 10 {
		require.NoErrorf(t, clickAsAccount(t, server, "1.2.3.4", boosted), "click %d", click)
	}
	clock.Advance(10 * time.Second)

	for click := range 20 {
		require.NoErrorf(t, clickAsAccount(t, server, "1.2.3.4", boosted), "click %d", click)
	}

	err := clickAsAccount(t, server, "1.2.3.4", boosted)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err),
		"thirty in the boosted account's hand, but its scope holds twenty")
}

func TestTheBudgetIsTheTighterBucket(t *testing.T) {
	server, _, _ := accountServer(t, clicks.ThrottleConfig{
		Config: cpratelimit.Config{PerSecond: 1, Burst: 10}, ScopeMultiplier: 2,
	})

	for click := range 15 {
		require.NoErrorf(t, clickAsAccount(t, server, "1.2.3.4", accountNumber(click%2)), "click %d", click)
	}

	read := func(token string) *planetv1.ClickBudget {
		t.Helper()

		req := connect.NewRequest(&planetv1.GetBudgetRequest{})
		req.Header().Set("X-Real-IP", "1.2.3.4")
		if token != "" {
			req.Header().Set(cpconnect.SessionHeader, token)
		}

		res, err := planetv1connect.NewClickServiceClient(server.Client(), server.URL).GetBudget(t.Context(), req)
		require.NoError(t, err)

		return res.Msg.GetBudget()
	}

	budget := read(accountNumber(2).String())
	require.InDelta(t, 5.0, budget.GetTokens(), 1e-9, "a fresh account holds ten, its scope five")
	require.Equal(t, uint32(20), budget.GetCapacity())
	require.InDelta(t, 2.0, budget.GetRefillPerSecond(), 1e-9)

	require.InDelta(t, 10.0, read("").GetTokens(), 1e-9, "no token reads the scope's bucket from before accounts")
	require.InDelta(t, 10.0, read("forged").GetTokens(), 1e-9, "a bad token is not refused on a read")
}

func TestALinkedAccountClicksTwiceAsFastAsAGuest(t *testing.T) {
	server, _, clock := accountServer(t, clicks.ThrottleConfig{Config: cpratelimit.Config{PerSecond: 1, Burst: 10}})
	guest, linked := accountNumber(0).String(), linkedPrefix+accountNumber(1).String()

	for click := range 10 {
		_, err := clickWithToken(t, server, "1.2.3.4", guest)
		require.NoErrorf(t, err, "guest click %d", click)
	}
	_, err := clickWithToken(t, server, "1.2.3.4", guest)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "a guest holds ten")
	require.InDelta(t, 2.0, budgetDetail(t, err).GetLinkedMultiplier(), 1e-9, "and is told what signing in is worth")

	for click := range 20 {
		_, err := clickWithToken(t, server, "5.6.7.8", linked)
		require.NoErrorf(t, err, "linked click %d", click)
	}
	_, err = clickWithToken(t, server, "5.6.7.8", linked)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "a linked account holds twenty")

	clock.Advance(time.Second)
	res, err := clickWithToken(t, server, "5.6.7.8", linked)
	require.NoError(t, err, "and refills two a second")
	require.Equal(t, uint32(20), res.Msg.GetBudget().GetCapacity())
	require.InDelta(t, 2.0, res.Msg.GetBudget().GetRefillPerSecond(), 1e-9)
	require.InDelta(t, 2.0, res.Msg.GetBudget().GetLinkedMultiplier(), 1e-9)
}

func TestSigningInDoesNotRefillTheGuestsBucket(t *testing.T) {
	server, _, _ := accountServer(t, clicks.ThrottleConfig{Config: cpratelimit.Config{PerSecond: 1, Burst: 10}})
	account := accountNumber(0).String()

	for click := range 10 {
		_, err := clickWithToken(t, server, "1.2.3.4", account)
		require.NoErrorf(t, err, "click %d", click)
	}

	// The linked bucket is another key, so it starts full: signing in is a one-time top-up, not a way to refill.
	res, err := clickWithToken(t, server, "1.2.3.4", linkedPrefix+account)
	require.NoError(t, err)
	require.InDelta(t, 19.0, res.Msg.GetBudget().GetTokens(), 1e-9)

	_, err = clickWithToken(t, server, "1.2.3.4", account)
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "the guest's bucket is still spent")
}
