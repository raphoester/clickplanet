package shadowban_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func newClock() *cptime.FixedClock {
	return cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))
}

const threeYears = 3 * 365 * 24 * time.Hour

// newBanner keeps its bans in memory and fails the test on any state error.
func newBanner(t *testing.T, config shadowban.Config, clock cptime.Clock) *shadowban.Banner {
	t.Helper()
	return shadowban.New(config, clock, shadowban.NewMemoryPersistence(), failOnStateError(t))
}

func failOnStateError(t *testing.T) func(error) {
	t.Helper()
	return func(err error) { assert.NoError(t, err, "unexpected state error") }
}

func config() shadowban.Config {
	return shadowban.Config{
		Enforce:        true,
		BanDurations:   []time.Duration{time.Hour, 24 * time.Hour, threeYears},
		ReflagInterval: 5 * time.Minute,
		SaveInterval:   time.Minute,
	}
}

func TestAFirstOffenceBansForTheFirstStep(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	require.False(t, banner.Banned("bot"))

	sentence, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 1, sentence.Flags)
	assert.Equal(t, 1, sentence.Offence)
	assert.Equal(t, clock.Now().Add(time.Hour), sentence.Until)

	clock.Advance(59 * time.Minute)
	assert.True(t, banner.Banned("bot"))

	clock.Advance(2 * time.Minute)
	assert.False(t, banner.Banned("bot"))
	assert.Equal(t, 0, banner.Flagged())
}

func TestEnforceOffBansNothingAndStillCounts(t *testing.T) {
	c := config()
	c.Enforce = false

	banner := newBanner(t, c, newClock())

	_, accepted := banner.Flag("bot")
	require.True(t, accepted)

	assert.False(t, banner.Banned("bot"))
	assert.Equal(t, 1, banner.Flagged())
	assert.False(t, banner.Enforcing())
}

func TestAFlagInsideTheReflagIntervalSaysNothingNew(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	_, accepted := banner.Flag("bot")
	require.True(t, accepted)

	clock.Advance(time.Minute)
	sentence, accepted := banner.Flag("bot")
	assert.False(t, accepted)
	assert.Equal(t, 1, sentence.Flags)

	clock.Advance(5 * time.Minute)
	sentence, accepted = banner.Flag("bot")
	assert.True(t, accepted)
	assert.Equal(t, 2, sentence.Flags)
}

func TestAReflagExtendsTheRunningBanWithoutANewOffence(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	banner.Flag("bot")

	for range 11 {
		clock.Advance(50 * time.Minute)
		sentence, accepted := banner.Flag("bot")
		require.True(t, accepted)
		require.Equal(t, 1, sentence.Offence, "a caller that never stops is one offence")
	}

	clock.Advance(30 * time.Minute)
	assert.True(t, banner.Banned("bot"))
}

func TestAnOffenceAfterALapsedBanClimbsTheLadder(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	banner.Flag("bot")
	clock.Advance(2 * time.Hour)
	require.False(t, banner.Banned("bot"))

	sentence, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 2, sentence.Offence)
	assert.Equal(t, 2, sentence.Flags)
	assert.Equal(t, clock.Now().Add(24*time.Hour), sentence.Until)

	clock.Advance(23 * time.Hour)
	assert.True(t, banner.Banned("bot"))
}

func TestTheThirdOffenceBansForThreeYears(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	banner.Flag("bot")
	clock.Advance(2 * time.Hour)
	banner.Flag("bot")
	clock.Advance(25 * time.Hour)

	sentence, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 3, sentence.Offence)
	assert.Equal(t, clock.Now().Add(threeYears), sentence.Until)

	clock.Advance(threeYears - time.Hour)
	assert.True(t, banner.Banned("bot"))

	clock.Advance(2 * time.Hour)
	assert.False(t, banner.Banned("bot"))
}

func TestTheLastStepRepeats(t *testing.T) {
	c := config()
	c.BanDurations = []time.Duration{time.Hour, 24 * time.Hour}

	clock := newClock()
	banner := newBanner(t, c, clock)

	var sentence shadowban.Sentence
	for range 5 {
		sentence, _ = banner.Flag("bot")
		clock.Advance(25 * time.Hour)
	}

	assert.Equal(t, 5, sentence.Offence)
	assert.Equal(t, clock.Now().Add(-time.Hour), sentence.Until)
}

func TestAnOffenceCountsForever(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	banner.Flag("bot")
	clock.Advance(10 * 365 * 24 * time.Hour)

	sentence, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 2, sentence.Offence)
}

func TestAnEmptyScopeIsNeverBanned(t *testing.T) {
	banner := newBanner(t, config(), newClock())

	_, accepted := banner.Flag("")
	assert.False(t, accepted)
	assert.Equal(t, 0, banner.Flagged())
}

func TestAManualBanTakesTheLadderAndCountsAsAnOffence(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	sentence := banner.Ban("bot", 0)
	assert.Equal(t, shadowban.Sentence{Offence: 1, Until: clock.Now().Add(time.Hour)}, sentence)
	assert.True(t, banner.Banned("bot"))

	clock.Advance(2 * time.Hour)
	_, running := banner.Sentence("bot")
	assert.False(t, running)

	sentence, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 2, sentence.Offence, "the next flag is a second offence")
}

func TestAManualBanWithADurationNeverShortensARunningOne(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	banner.Ban("bot", 48*time.Hour)
	sentence := banner.Ban("bot", time.Minute)

	assert.Equal(t, 1, sentence.Offence, "a ban on a running ban extends it, it is not a new offence")
	assert.Equal(t, clock.Now().Add(48*time.Hour), sentence.Until)

	running, ok := banner.Sentence("bot")
	require.True(t, ok)
	assert.Equal(t, sentence, running)
}

func TestAManualBanOnAnEmptyScopeBansNothing(t *testing.T) {
	banner := newBanner(t, config(), newClock())

	assert.Equal(t, shadowban.Sentence{}, banner.Ban("", time.Hour))
	assert.Equal(t, 0, banner.Flagged())
}
