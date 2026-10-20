package bans_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func newClock() *cptime.FixedClock {
	return cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))
}

const threeYears = 3 * 365 * 24 * time.Hour

func newBanner(t *testing.T, config bans.Config, clock cptime.Clock) banner {
	t.Helper()
	return bannerOver(t, config, clock, bans.NewMemoryStore())
}

func bannerOver(t *testing.T, config bans.Config, clock cptime.Clock, store bans.Store) banner {
	t.Helper()
	return banner{t: t, inner: bans.New(config, clock, store)}
}

func config() bans.Config {
	return bans.Config{
		Enforce:        true,
		BanDurations:   []time.Duration{time.Hour, 24 * time.Hour, threeYears},
		ReflagInterval: 5 * time.Minute,
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

	var sentence bans.Sentence
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
	assert.Equal(t, bans.Sentence{Offence: 1, Until: clock.Now().Add(time.Hour)}, sentence)
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

	assert.Equal(t, bans.Sentence{}, banner.Ban("", time.Hour))
	assert.Equal(t, 0, banner.Flagged())
}

func TestAnUnbanEndsTheBanNowAndForgetsItsOffence(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	banner.Flag("player")
	clock.Advance(10 * time.Minute)
	banner.Unban("player")

	assert.False(t, banner.Banned("player"))
	_, running := banner.Sentence("player")
	assert.False(t, running)
	assert.Equal(t, 0, banner.Flagged())

	sentence, accepted := banner.Flag("player")
	require.True(t, accepted)
	assert.Equal(t, 1, sentence.Offence, "the next ban starts the ladder where the lifted one did")
	assert.Equal(t, clock.Now().Add(time.Hour), sentence.Until)
}

func TestAnUnbanForgetsOnlyTheOffenceItLifts(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	banner.Ban("bot", 0)
	clock.Advance(2 * time.Hour)
	banner.Ban("bot", 0)
	banner.Unban("bot")

	sentence := banner.Ban("bot", 0)
	assert.Equal(t, 2, sentence.Offence, "the first offence was not the one lifted")
	assert.Equal(t, clock.Now().Add(24*time.Hour), sentence.Until)
}

func TestAnUnbanOfAnEndedBanForgetsItsOffenceAndKeepsItsEnd(t *testing.T) {
	clock := newClock()
	store := bans.NewMemoryStore()
	banner := bannerOver(t, config(), clock, store)

	banned, _ := banner.Flag("player")
	clock.Advance(2 * time.Hour)
	banner.Unban("player")

	assert.Equal(t, 0, store.Stored()["player"].Offences)
	assert.Equal(t, banned.Until, store.Stored()["player"].ExpiresAt, "a ban that ended is not moved")

	sentence, accepted := banner.Flag("player")
	require.True(t, accepted)
	assert.Equal(t, 1, sentence.Offence, "the next ban starts the ladder where the lifted one did")
	assert.Equal(t, clock.Now().Add(time.Hour), sentence.Until)
}

func TestEachUnbanForgetsOneOffence(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	banner.Ban("player", 0)
	clock.Advance(2 * time.Hour)
	banner.Ban("player", 0)

	banner.Unban("player")
	assert.False(t, banner.Banned("player"))
	last, found := banner.LastSentence("player")
	require.True(t, found, "the first offence still counts")
	assert.Equal(t, 1, last.Offence)

	banner.Unban("player")
	_, found = banner.LastSentence("player")
	assert.False(t, found)

	assert.Equal(t, 1, banner.Ban("player", 0).Offence)
}

func TestAnUnbanWithNoBanChangesNothing(t *testing.T) {
	clock := newClock()
	store := bans.NewMemoryStore()
	banner := bannerOver(t, config(), clock, store)

	banner.Flag("lifted")
	banner.Unban("lifted")
	before := store.Stored()

	banner.Unban("lifted")
	banner.Unban("nobody")
	banner.Unban("")

	assert.Equal(t, before, store.Stored(), "no offence is left, so nothing is written")
}

func TestTheLastSentenceIsTheLatestBanStillCountedRunningOrEnded(t *testing.T) {
	clock := newClock()
	banner := newBanner(t, config(), clock)

	_, found := banner.LastSentence("nobody")
	assert.False(t, found)
	_, found = banner.LastSentence("")
	assert.False(t, found)

	banned, _ := banner.Flag("player")

	last, found := banner.LastSentence("player")
	require.True(t, found)
	assert.Equal(t, banned, last)

	clock.Advance(2 * time.Hour)
	_, running := banner.Sentence("player")
	require.False(t, running)

	last, found = banner.LastSentence("player")
	require.True(t, found, "an ended ban is still an offence")
	assert.Equal(t, banned, last)
}
