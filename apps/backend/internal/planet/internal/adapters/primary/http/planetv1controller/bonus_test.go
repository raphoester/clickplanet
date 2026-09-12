package planetv1controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cphttpserver"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBonuses struct {
	reward    bonus.Reward
	claimable bool

	claimedToken string
	claimedScope string
	published    []bonus.Taken
	events       chan bonus.Event
}

func (s *stubBonuses) Attend(string) (<-chan bonus.Event, func()) {
	if s.events == nil {
		s.events = make(chan bonus.Event, 4)
	}
	return s.events, func() {}
}

func (s *stubBonuses) Claim(token string, scope string) (bonus.Reward, bool) {
	s.claimedToken, s.claimedScope = token, scope
	if !s.claimable {
		return bonus.Reward{}, false
	}
	return s.reward, true
}

func (s *stubBonuses) Clicked(string) {}

func (s *stubBonuses) Publish(taken bonus.Taken) { s.published = append(s.published, taken) }

func (s *stubBonuses) Multiplier() float64 { return 3 }

type stubBooster struct {
	key        string
	multiplier float64
	until      time.Time
	state      cpratelimit.State
}

func (s *stubBooster) Boost(key string, multiplier float64, until time.Time) cpratelimit.State {
	s.key, s.multiplier, s.until = key, multiplier, until
	return s.state
}

type frozenClock struct{ now time.Time }

func (c frozenClock) Now() time.Time { return c.now }

var bonusEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func newBonusClient(t *testing.T, bonuses BonusRegistry, booster ClickBooster) planetv1connect.ClickServiceClient {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(planetv1connect.NewClickServiceHandler(
		NewClickService(
			stubService{}, stubChecker{}, stubMapReader{}, stubSubscriber{},
			DefaultHeartbeat, nil, bonuses, booster, frozenClock{now: bonusEpoch},
		),
		connect.WithInterceptors(NewErrorInterceptor(nil)),
	))

	server := httptest.NewServer(cphttpserver.IPReaderMiddleware(mux))
	t.Cleanup(server.Close)

	return planetv1connect.NewClickServiceClient(server.Client(), server.URL)
}

func claim(t *testing.T, client planetv1connect.ClickServiceClient, token string) (*planetv1.ClaimBonusResponse, error) {
	t.Helper()

	res, err := client.ClaimBonus(t.Context(), connect.NewRequest(&planetv1.ClaimBonusRequest{
		Token:     token,
		CountryId: "fr",
	}))
	if err != nil {
		return nil, fmt.Errorf("claim refused: %w", err)
	}

	return res.Msg, nil
}

func TestClaimingABonusStartsTheBoost(t *testing.T) {
	bonuses := &stubBonuses{claimable: true, reward: bonus.Reward{Kind: bonus.KindTripleClicks, Duration: time.Minute}}
	booster := &stubBooster{state: cpratelimit.State{Tokens: 4, Capacity: 30, PerSecond: 3}}

	msg, err := claim(t, newBonusClient(t, bonuses, booster), "a-token")
	require.NoError(t, err)

	assert.Equal(t, "a-token", bonuses.claimedToken)
	assert.InDelta(t, 3.0, booster.multiplier, 1e-9)
	assert.Equal(t, bonusEpoch.Add(time.Minute), booster.until)
	assert.Equal(t, planetv1.BonusKind_BONUS_KIND_TRIPLE_CLICKS, msg.GetKind())
	assert.Equal(t, uint32(60), msg.GetDurationSeconds())
}

func TestTheAnswerCarriesTheWidenedAllowance(t *testing.T) {
	// So the meter widens on the answer to the claim rather than waiting for
	// the player's next click to learn the new policy.
	bonuses := &stubBonuses{claimable: true, reward: bonus.Reward{Kind: bonus.KindTripleClicks, Duration: time.Minute}}
	booster := &stubBooster{state: cpratelimit.State{Tokens: 7, Capacity: 30, PerSecond: 3}}

	msg, err := claim(t, newBonusClient(t, bonuses, booster), "a-token")
	require.NoError(t, err)

	assert.Equal(t, uint32(30), msg.GetBudget().GetCapacity())
	assert.InDelta(t, 3.0, msg.GetBudget().GetRefillPerSecond(), 1e-9)
	assert.InDelta(t, 7.0, msg.GetBudget().GetTokens(), 1e-9)
}

func TestTheBoostIsKeyedOnTheSameScopeTheThrottleIs(t *testing.T) {
	// The caller drawn for the offer, the one allowed to claim it and the
	// bucket that gets widened have to be one and the same, or a bonus widens
	// somebody else's allowance.
	bonuses := &stubBonuses{claimable: true, reward: bonus.Reward{Kind: bonus.KindTripleClicks, Duration: time.Minute}}
	booster := &stubBooster{}

	_, err := claim(t, newBonusClient(t, bonuses, booster), "a-token")
	require.NoError(t, err)

	assert.Equal(t, bonuses.claimedScope, booster.key)
}

func TestACatchIsAnnouncedWithTheCountryTheClaimNamed(t *testing.T) {
	bonuses := &stubBonuses{claimable: true, reward: bonus.Reward{Kind: bonus.KindTripleClicks, Duration: time.Minute}}

	_, err := claim(t, newBonusClient(t, bonuses, &stubBooster{}), "a-token")
	require.NoError(t, err)

	require.Len(t, bonuses.published, 1)
	assert.Equal(t, "fr", bonuses.published[0].CountryID)
}

func TestARefusedClaimIsNotFoundAndBoostsNothing(t *testing.T) {
	bonuses := &stubBonuses{claimable: false}
	booster := &stubBooster{}

	_, err := claim(t, newBonusClient(t, bonuses, booster), "someone-elses-token")

	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.Zero(t, booster.multiplier, "a refused claim must not widen anything")
	assert.Empty(t, bonuses.published, "and must not announce a catch that did not happen")
}

func TestARefusedClaimSaysNothingAboutWhy(t *testing.T) {
	// Unknown, spent, lapsed and somebody else's all answer the same. The
	// difference between them is exactly what a script guessing tokens would
	// measure.
	_, err := claim(t, newBonusClient(t, &stubBonuses{claimable: false}, &stubBooster{}), "nope")

	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrNoSuchBonus.Error())
}

func TestWithBoxesOffTheProcedureIsUnimplemented(t *testing.T) {
	// The same shape chat being disabled has: the capability is absent rather
	// than present and failing.
	_, err := claim(t, newBonusClient(t, nil, nil), "a-token")

	require.Error(t, err)
	assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestAnOfferReachesTheStreamAsItsOwnEvent(t *testing.T) {
	bonuses := &stubBonuses{events: make(chan bonus.Event, 4)}
	client := newBonusClient(t, bonuses, &stubBooster{})

	bonuses.events <- bonus.Event{Offer: &bonus.Offer{
		Token:     "a-token",
		Seed:      42,
		Kind:      bonus.KindTripleClicks,
		Duration:  time.Minute,
		ExpiresAt: bonusEpoch.Add(15 * time.Second),
	}}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	stream, err := client.ListenForEvents(ctx, connect.NewRequest(&planetv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	require.True(t, stream.Receive(), "the stream closed before sending anything")

	offered := stream.Msg().GetBonusOffered()
	require.NotNil(t, offered, "expected a bonus_offered case, got %+v", stream.Msg().GetEvent())
	assert.Equal(t, "a-token", offered.GetToken())
	assert.Equal(t, uint32(42), offered.GetSeed())
	assert.Equal(t, uint32(60), offered.GetDurationSeconds())
	assert.Equal(t, bonusEpoch.Add(15*time.Second).UnixMilli(), offered.GetExpiresAtUnixMs())
}

func TestACatchReachesTheStreamWithNoTokenOnIt(t *testing.T) {
	bonuses := &stubBonuses{events: make(chan bonus.Event, 4)}
	client := newBonusClient(t, bonuses, &stubBooster{})

	bonuses.events <- bonus.Event{Taken: &bonus.Taken{CountryID: "jp", Kind: bonus.KindTripleClicks}}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	stream, err := client.ListenForEvents(ctx, connect.NewRequest(&planetv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	require.True(t, stream.Receive())

	taken := stream.Msg().GetBonusTaken()
	require.NotNil(t, taken)
	assert.Equal(t, "jp", taken.GetCountryId())
}

func TestAnEmptyBonusEventIsSkippedRatherThanSent(t *testing.T) {
	// A case added to bonus.Event and forgotten in the encoder drops here
	// rather than becoming a frame that says nothing.
	assert.Nil(t, bonusEvent(bonus.Event{}))
}
