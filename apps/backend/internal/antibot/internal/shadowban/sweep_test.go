package shadowban

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepKeepsACallerServingABan(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	b := New(Config{Enforce: true, BanDuration: 24 * time.Hour, ReflagInterval: 5 * time.Minute}, clock)

	b.Flag("bot")
	require.Equal(t, 1, b.Flagged())

	clock.Advance(2 * time.Hour)
	b.sweep()

	assert.Contains(t, b.bans, "bot", "a sweep must not release a ban still running")
	assert.Equal(t, 1, b.Flagged())
}

func TestSweepForgetsABanNothingWouldStillPrint(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	b := New(Config{Enforce: true, BanDuration: time.Minute, ReflagInterval: time.Minute}, clock)

	b.Flag("bot")
	require.Len(t, b.bans, 1)

	clock.Advance(2 * time.Minute)
	b.sweep()

	assert.Empty(t, b.bans)
}

func TestSweepKeepsALapsedBanWhileItsFlagCountStillMeansSomething(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	b := New(Config{Enforce: true, BanDuration: time.Minute, ReflagInterval: time.Hour}, clock)

	b.Flag("bot")

	clock.Advance(2 * time.Minute)
	b.sweep()

	require.Contains(t, b.bans, "bot", "forgetting here would reset the count to one")

	flags, accepted := b.Flag("bot")
	assert.False(t, accepted)
	assert.Equal(t, 1, flags)
}
