package shadowban_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/shadowban"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

func newClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}
}

func config() shadowban.Config {
	return shadowban.Config{
		Enforce:        true,
		BanDuration:    time.Hour,
		ReflagInterval: 5 * time.Minute,
		SweepInterval:  time.Minute,
	}
}

func TestAFlagBansForItsDuration(t *testing.T) {
	clock := newClock()
	banner := shadowban.New(config(), clock)

	require.False(t, banner.Banned("bot"))

	flags, accepted := banner.Flag("bot")
	require.True(t, accepted)
	assert.Equal(t, 1, flags)

	assert.True(t, banner.Banned("bot"))
	assert.Equal(t, 1, banner.Flagged())

	clock.advance(59 * time.Minute)
	assert.True(t, banner.Banned("bot"))

	clock.advance(2 * time.Minute)
	assert.False(t, banner.Banned("bot"), "the ban lapses on its own")
	assert.Equal(t, 0, banner.Flagged())
}

func TestEnforceOffBansNothingAndStillCounts(t *testing.T) {
	c := config()
	c.Enforce = false

	banner := shadowban.New(c, newClock())

	_, accepted := banner.Flag("bot")
	require.True(t, accepted, "enforce must not change what is judged")

	assert.False(t, banner.Banned("bot"), "only enforcing drops clicks")
	assert.Equal(t, 1, banner.Flagged(), "the gauge answers what enforcing would cost")
	assert.False(t, banner.Enforcing())
}

func TestAFlagInsideTheReflagIntervalSaysNothingNew(t *testing.T) {
	clock := newClock()
	banner := shadowban.New(config(), clock)

	_, accepted := banner.Flag("bot")
	require.True(t, accepted)

	clock.advance(time.Minute)
	flags, accepted := banner.Flag("bot")
	assert.False(t, accepted, "the caller is already serving this one")
	assert.Equal(t, 1, flags)

	clock.advance(5 * time.Minute)
	flags, accepted = banner.Flag("bot")
	assert.True(t, accepted, "past the interval it is a fresh judgement")
	assert.Equal(t, 2, flags, "a rising count is independent evidence repeating")
}

func TestABanIsExtendedByEachNewFlag(t *testing.T) {
	clock := newClock()
	banner := shadowban.New(config(), clock)

	banner.Flag("bot")

	clock.advance(50 * time.Minute)
	_, accepted := banner.Flag("bot")
	require.True(t, accepted)

	clock.advance(30 * time.Minute)
	assert.True(t, banner.Banned("bot"), "the second flag carries its own hour")
}

func TestAnEmptyScopeIsNeverBanned(t *testing.T) {
	banner := shadowban.New(config(), newClock())

	_, accepted := banner.Flag("")
	assert.False(t, accepted)
	assert.Equal(t, 0, banner.Flagged())
}
