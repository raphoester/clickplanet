package shadowban_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
)

func TestTheBansAreTheStoresAndNotTheBanners(t *testing.T) {
	clock := newClock()
	store := shadowban.NewMemoryStore()

	before := bannerOver(t, config(), clock, store)
	before.Flag("repeat")
	clock.Advance(2 * time.Hour)
	before.Flag("repeat")
	clock.Advance(25 * time.Hour)
	before.Flag("repeat")
	before.Flag("fresh")

	after := bannerOver(t, config(), clock, store)

	assert.True(t, after.Banned("repeat"))
	assert.True(t, after.Banned("fresh"))

	clock.Advance(2 * time.Hour)
	assert.True(t, after.Banned("repeat"), "a three-year ban outlives the banner that passed it")
	assert.False(t, after.Banned("fresh"), "the first offence still lapses on time")

	sentence, accepted := after.Flag("fresh")
	require.True(t, accepted)
	assert.Equal(t, 2, sentence.Offence, "the offence count was kept")
}

func TestTheReflagIntervalIsKeptInTheStore(t *testing.T) {
	clock := newClock()
	store := shadowban.NewMemoryStore()

	bannerOver(t, config(), clock, store).Flag("bot")

	clock.Advance(time.Minute)
	_, accepted := bannerOver(t, config(), clock, store).Flag("bot")
	assert.False(t, accepted, "a restart is no way around the reflag interval")

	clock.Advance(5 * time.Minute)
	_, accepted = bannerOver(t, config(), clock, store).Flag("bot")
	assert.True(t, accepted)
}

func TestABanInTheStoreIsRunning(t *testing.T) {
	clock := newClock()
	store := shadowban.NewMemoryStore(shadowban.Record{Key: "kept", Flags: 1, Offences: 1, ExpiresAt: clock.Now().Add(time.Hour)})

	banner := bannerOver(t, config(), clock, store)

	assert.True(t, banner.Banned("kept"))
	assert.False(t, banner.Banned("other"))
	assert.Equal(t, 1, banner.Flagged())
}

func TestAFailingStoreIsAnError(t *testing.T) {
	store := shadowban.NewMemoryStore()
	cause := errors.New("postgres is down")
	store.FailWith(cause)
	banner := shadowban.New(config(), newClock(), store)
	ctx := t.Context()

	_, _, err := banner.Flag(ctx, "bot")
	require.ErrorIs(t, err, cause)
	_, err = banner.Ban(ctx, "bot", 0)
	require.ErrorIs(t, err, cause)
	require.ErrorIs(t, banner.Unban(ctx, "bot"), cause)
	_, _, err = banner.Sentence(ctx, "bot")
	require.ErrorIs(t, err, cause)
	_, err = banner.Banned(ctx, "bot")
	require.ErrorIs(t, err, cause)
	_, err = banner.Flagged(ctx)
	require.ErrorIs(t, err, cause)
}

func TestWithEnforceOffBannedReadsNothing(t *testing.T) {
	store := shadowban.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))
	c := config()
	c.Enforce = false

	banned, err := shadowban.New(c, newClock(), store).Banned(t.Context(), "bot")

	require.NoError(t, err)
	assert.False(t, banned)
}

func TestAnUnbanIsKeptInTheStore(t *testing.T) {
	clock := newClock()
	store := shadowban.NewMemoryStore()
	flaggedAt := clock.Now()

	before := bannerOver(t, config(), clock, store)
	before.Flag("player")
	clock.Advance(10 * time.Minute)
	before.Unban("player")

	assert.Equal(t, shadowban.Record{
		Key: "player", Flags: 1, ExpiresAt: clock.Now(), LastFlaggedAt: flaggedAt,
	}, store.Stored()["player"])

	after := bannerOver(t, config(), clock, store)
	assert.False(t, after.Banned("player"))

	sentence, accepted := after.Flag("player")
	require.True(t, accepted)
	assert.Equal(t, 1, sentence.Offence)
}
