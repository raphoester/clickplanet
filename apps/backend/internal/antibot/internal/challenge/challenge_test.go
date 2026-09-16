package challenge_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/challenge"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type register struct {
	challenges *challenge.Challenges
	clock      *cptime.FixedClock
	answered   []string
}

func newRegister(config challenge.Config) *register {
	r := &register{clock: cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))}

	r.challenges = challenge.New(config, r.clock, challenge.Hooks{
		OnAnswered: func(scope string) { r.answered = append(r.answered, scope) },
	})

	return r
}

func enforced() challenge.Config {
	return challenge.Config{Enforce: true, MinSuspects: 1, Interval: 10 * time.Minute}
}

func TestAChallengeStandsUntilItIsAnswered(t *testing.T) {
	r := newRegister(enforced())

	require.True(t, r.challenges.Raise("scope", "mint-1"))

	for range 5 {
		r.clock.Advance(time.Second)
		assert.True(t, r.challenges.Standing("scope", "mint-1"),
			"the caller is still holding the session it was challenged on")
	}

	assert.Equal(t, 1, r.challenges.Count())
	assert.Empty(t, r.answered)
}

func TestComingBackWithAFreshSessionAnswersIt(t *testing.T) {
	r := newRegister(enforced())

	require.True(t, r.challenges.Raise("scope", "mint-1"))
	require.True(t, r.challenges.Standing("scope", "mint-1"))

	r.clock.Advance(2 * time.Second)

	assert.False(t, r.challenges.Standing("scope", "mint-2"),
		"the only way to a new session is the mint, and the only way through the mint is Turnstile")
	assert.Equal(t, []string{"scope"}, r.answered)
	assert.Equal(t, 0, r.challenges.Count())
}

func TestAnAnsweredChallengeIsNotRaisedAgainInsideTheInterval(t *testing.T) {
	r := newRegister(enforced())

	require.True(t, r.challenges.Raise("scope", "mint-1"))
	require.False(t, r.challenges.Standing("scope", "mint-2"))

	r.clock.Advance(9 * time.Minute)
	assert.False(t, r.challenges.Raise("scope", "mint-2"), "or the player is put straight back in front of the widget")
	assert.False(t, r.challenges.Standing("scope", "mint-2"))

	r.clock.Advance(2 * time.Minute)
	assert.True(t, r.challenges.Raise("scope", "mint-2"), "past the interval the next one rests on newer readings")
	assert.Len(t, r.answered, 1, "one pass, not two")
}

func TestAnUnansweredChallengeLapsesRatherThanStandingForever(t *testing.T) {
	r := newRegister(enforced())

	require.True(t, r.challenges.Raise("scope", "mint-1"))

	r.clock.Advance(10 * time.Minute)

	assert.False(t, r.challenges.Standing("scope", "mint-1"),
		"a client that cannot answer one is refused for an interval, not for good")
	assert.Equal(t, 0, r.challenges.Count())
}

func TestDroppingTheTokenIsNotAnAnswer(t *testing.T) {
	r := newRegister(enforced())

	require.True(t, r.challenges.Raise("scope", "mint-1"))

	assert.True(t, r.challenges.Standing("scope", ""), "coming back with no session proves nothing")
	assert.Empty(t, r.answered)
}

func TestACallerWithNoSessionIsNeverChallenged(t *testing.T) {
	r := newRegister(enforced())

	assert.False(t, r.challenges.Raise("scope", ""),
		"there is no fresh session it could come back with, so the challenge would never clear")
	assert.False(t, r.challenges.Raise("", "mint-1"))
	assert.Equal(t, 0, r.challenges.Count())
}

func TestWithEnforceOffTheChallengeIsCountedAndNothingIsRefused(t *testing.T) {
	config := enforced()
	config.Enforce = false

	r := newRegister(config)

	require.True(t, r.challenges.Raise("scope", "mint-1"), "still recorded: counting what it would refuse is the point")
	assert.False(t, r.challenges.Standing("scope", "mint-1"))
	assert.Equal(t, 1, r.challenges.Count(), "and still visible on the gauge")
}
