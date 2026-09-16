package challenge

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheSweepForgetsLapsedRecords(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))

	c := New(Config{Enforce: true, MinSuspects: 1, Interval: 10 * time.Minute}, clock, Hooks{})

	require.True(t, c.Raise("standing", "mint-1"))
	require.True(t, c.Raise("lapsing", "mint-1"))

	clock.Advance(6 * time.Minute)
	require.True(t, c.Raise("newer", "mint-1"))

	clock.Advance(5 * time.Minute)
	c.forget(clock.Now())

	assert.Equal(t, []string{"newer"}, keys(c),
		"without this the map keeps an entry for every caller ever challenged")
}

func keys(c *Challenges) []string {
	out := make([]string, 0, len(c.challenges))
	for scope := range c.challenges {
		out = append(out, scope)
	}
	return out
}
