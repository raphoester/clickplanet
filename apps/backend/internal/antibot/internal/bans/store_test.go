package bans_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
)

func TestTheBansAreTheStoresAndNotTheBanners(t *testing.T) {
	clock := newClock()
	store := bans.NewMemoryStore()

	before := bannerOver(t, ladder(), clock, store)
	before.Flag("repeat")
	clock.Advance(2 * time.Hour)
	before.Flag("repeat")
	clock.Advance(25 * time.Hour)
	before.Flag("repeat")
	before.Flag("fresh")

	after := bannerOver(t, ladder(), clock, store)

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
	store := bans.NewMemoryStore()

	bannerOver(t, ladder(), clock, store).Flag("bot")

	clock.Advance(time.Minute)
	_, accepted := bannerOver(t, ladder(), clock, store).Flag("bot")
	assert.False(t, accepted, "a restart is no way around the reflag interval")

	clock.Advance(5 * time.Minute)
	_, accepted = bannerOver(t, ladder(), clock, store).Flag("bot")
	assert.True(t, accepted)
}

func TestABanInTheStoreIsRunning(t *testing.T) {
	clock := newClock()
	store := bans.NewMemoryStore(bans.Record{Key: "kept", Flags: 1, Offences: 1, ExpiresAt: clock.Now().Add(time.Hour)})

	banner := bannerOver(t, ladder(), clock, store)

	assert.True(t, banner.Banned("kept"))
	assert.False(t, banner.Banned("other"))
	assert.Equal(t, 1, banner.Flagged())
}

func TestAnUnbanIsKeptInTheStore(t *testing.T) {
	clock := newClock()
	store := bans.NewMemoryStore()
	flaggedAt := clock.Now()

	before := bannerOver(t, ladder(), clock, store)
	before.Flag("player")
	clock.Advance(10 * time.Minute)
	before.Unban("player")

	assert.Equal(t, bans.Record{
		Key: "player", Flags: 1, ExpiresAt: clock.Now(), LastFlaggedAt: flaggedAt,
		Pardon: bans.Pardon{
			At:     clock.Now(),
			Lifted: bans.Sentence{Flags: 1, Offence: 1, Until: flaggedAt.Add(time.Hour)},
			Until:  clock.Now().Add(pardon),
		},
	}, store.Stored()["player"])

	after := bannerOver(t, ladder(), clock, store)
	assert.False(t, after.Banned("player"))
	_, accepted := after.Flag("player")
	assert.False(t, accepted, "the pardon is in the store, not in the banner")

	clock.Advance(pardon)
	sentence, accepted := after.Flag("player")
	require.True(t, accepted)
	assert.Equal(t, 1, sentence.Offence)
}
