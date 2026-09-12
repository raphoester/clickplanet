package claim_bonus_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/claim_bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

type stubRegistry struct {
	reward    bonus.Reward
	claimable bool

	token     string
	scope     string
	published []bonus.Taken
}

func (s *stubRegistry) Claim(token string, scope string) (bonus.Reward, bool) {
	s.token, s.scope = token, scope
	if !s.claimable {
		return bonus.Reward{}, false
	}

	return s.reward, true
}

func (s *stubRegistry) Publish(taken bonus.Taken) { s.published = append(s.published, taken) }

func (s *stubRegistry) Multiplier() float64 { return 3 }

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

func granted() *stubRegistry {
	return &stubRegistry{claimable: true, reward: bonus.Reward{Kind: bonus.KindTripleClicks, Duration: time.Minute}}
}

func TestAClaimStartsTheBoostForTheDurationGranted(t *testing.T) {
	booster := &stubBooster{state: cpratelimit.State{Capacity: 30, PerSecond: 3}}

	out, err := claim_bonus.New(granted(), booster, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus.In{Token: "a-token", CountryID: "fr"})
	require.NoError(t, err)

	assert.InDelta(t, 3.0, booster.multiplier, 1e-9)
	assert.Equal(t, epoch.Add(time.Minute), booster.until)
	assert.Equal(t, 30, out.Budget.Capacity)
	assert.Equal(t, time.Minute, out.Duration)
}

func TestTheClaimAndTheBoostUseTheSameScope(t *testing.T) {
	registry, booster := granted(), &stubBooster{}

	_, err := claim_bonus.New(registry, booster, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus.In{Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, registry.scope, booster.key)
	assert.Equal(t, cpctx.RateLimitKey(t.Context()), booster.key)
}

func TestACatchIsAnnouncedWithTheCountryTheClaimNamed(t *testing.T) {
	registry := granted()

	_, err := claim_bonus.New(registry, &stubBooster{}, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus.In{Token: "a-token", CountryID: "jp"})
	require.NoError(t, err)

	require.Len(t, registry.published, 1)
	assert.Equal(t, "jp", registry.published[0].CountryID)
}

func TestARefusedClaimBoostsNothingAndAnnouncesNothing(t *testing.T) {
	registry, booster := &stubRegistry{claimable: false}, &stubBooster{}

	_, err := claim_bonus.New(registry, booster, cptime.NewFixedClock(epoch)).
		Execute(t.Context(), claim_bonus.In{Token: "not-mine"})

	require.ErrorIs(t, err, claim_bonus.ErrNoSuchBonus)
	assert.Zero(t, booster.multiplier)
	assert.Empty(t, registry.published)
}
