package shadowban

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func newSweepBanner(config Config) (*Banner, *cptime.FixedClock) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))
	config.Enforce = true
	return New(config, clock, nil), clock
}

func TestSweepKeepsACallerServingABan(t *testing.T) {
	b, clock := newSweepBanner(Config{BanDurations: []time.Duration{24 * time.Hour}})

	b.Flag("bot")

	clock.Advance(2 * time.Hour)
	b.sweep()

	assert.Contains(t, b.bans, "bot")
	assert.Equal(t, 1, b.Flagged())
}

func TestSweepKeepsAServedBanInsideTheStrikeMemory(t *testing.T) {
	b, clock := newSweepBanner(Config{
		BanDurations: []time.Duration{time.Minute},
		StrikeMemory: 24 * time.Hour,
	})

	b.Flag("bot")

	clock.Advance(time.Hour)
	b.sweep()

	require.Contains(t, b.bans, "bot", "forgetting here would make the next offence a first")

	sentence, accepted := b.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 2, sentence.Offence)
}

func TestSweepForgetsAServedBanPastTheStrikeMemory(t *testing.T) {
	b, clock := newSweepBanner(Config{
		BanDurations:   []time.Duration{time.Minute},
		StrikeMemory:   time.Hour,
		ReflagInterval: time.Minute,
	})

	b.Flag("bot")

	clock.Advance(2 * time.Hour)
	b.sweep()

	assert.Empty(t, b.bans)
}
