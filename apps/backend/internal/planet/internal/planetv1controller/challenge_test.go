package planetv1controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/antibot_challenge_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/throttle_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type stubChallenger struct{ challenged bool }

func (s *stubChallenger) Challenged(string, string) bool { return s.challenged }

// challengedServer wires the challenge where it lives — outside the throttle,
// inside everything the edge refuses — over the same served handler the other
// chain tests use.
func challengedServer(t *testing.T, guard antibot_challenge_click.ChallengeGuard) (*httptest.Server, *fakeLimiter) {
	t.Helper()

	limiter := &fakeLimiter{allow: true, state: cpratelimit.State{Tokens: 9, Capacity: 10, PerSecond: 1}}
	chain := antibot_challenge_click.New(throttle_click.New(stubService{}, limiter, onePrice), guard)

	return clickServerWith(t, chain, nil, connect.WithInterceptors(errorNet())), limiter
}

func TestTheChallengeRunsBeforeTheThrottle(t *testing.T) {
	server, limiter := challengedServer(t, &stubChallenger{challenged: true})

	_, err := clickAs(t, server, "1.2.3.4")

	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.Empty(t, limiter.keys,
		"a click refused because the caller must prove itself again must not also spend a token, "+
			"or the retry that follows the mint comes back 429 and the web app shows the throttle dialog")
}

func TestAnUnchallengedClickStillSpendsItsToken(t *testing.T) {
	server, limiter := challengedServer(t, &stubChallenger{})

	_, err := clickAs(t, server, "1.2.3.4")

	require.NoError(t, err)
	assert.Equal(t, []string{"1.2.3.4"}, limiter.keys)
}

func TestAChallengeReachesABrowserAsA401(t *testing.T) {
	server, _ := challengedServer(t, &stubChallenger{challenged: true})

	assert.Equal(t, http.StatusUnauthorized, clickStatus(t, server, "1.2.3.4"),
		"the code a client already answers by minting and retrying, which is what answers a challenge")
}

func TestTheChallengeCarriesNoBudget(t *testing.T) {
	server, _ := challengedServer(t, &stubChallenger{challenged: true})

	_, err := clickAs(t, server, "1.2.3.4")

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)

	for _, detail := range connectErr.Details() {
		value, valueErr := detail.Value()
		require.NoError(t, valueErr)

		_, isBudget := value.(*planetv1.ClickBudget)
		assert.False(t, isBudget, "no token was spent, so there is no new reading to report")
	}
}

// The mint's own throttle is one every 30s with ten in hand, and a challenge
// can be raised at most once per antiBot.challenge.interval. At the shipped ten
// minutes the bucket refills twenty times over between two challenges, so a
// player answering every one of them is never throttled out of doing it.
func TestAnsweringEveryChallengeDoesNotExhaustTheMintBudget(t *testing.T) {
	const interval = 10 * time.Minute

	clock := cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))
	mints := cpratelimit.New("mint-limiter", cpratelimit.Config{PerSecond: 1.0 / 30, Burst: 10}, clock)

	for range 50 {
		allowed, _ := mints.Take("1.2.3.4")
		require.True(t, allowed, "every challenge is answerable")

		clock.Advance(interval)
	}
}
