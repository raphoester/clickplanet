package rotator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSweepForgetsSlotsOlderThanTheWindow(t *testing.T) {
	clock := newClock()

	hour := bounded()
	hour.Window = time.Hour
	w := newWatchdog(Config{}, hour, clock)
	play(w, clock, 2, inTurns)

	clock.Advance(55 * time.Minute)
	w.sweep()
	require.Contains(t, w.scopes, scopeID(scope))
	assert.Len(t, w.scopes[scope].accounts["one"].slots, 1)

	clock.Advance(5 * time.Minute)
	w.sweep()
	assert.Empty(t, w.scopes, "an account with no slot left is forgotten, and so is its scope")
}

func TestSweepReportsEachPairChanceCouldHaveShown(t *testing.T) {
	for name, tc := range map[string]struct {
		turn  func(second int) []string
		share float64
	}{
		"one hand shares no second":          {turn: inTurns, share: 0},
		"two people share what chance gives": {turn: sideBySide, share: 5 / (40 * 40 / 300.0)},
	} {
		t.Run(name, func(t *testing.T) {
			clock := newClock()

			var shares []float64
			unbounded := func() Rules { return Rules{Window: 24 * time.Hour, MinSeconds: 15, MinByChance: 90, MaxShare: 0.05} }
			w := New(Config{}, unbounded, clock, func(share float64) { shares = append(shares, share) })

			play(w, clock, 10, tc.turn)
			w.sweep()
			assert.Empty(t, shares, "chance has not had its 90 seconds yet")

			play(w, clock, 10, tc.turn)
			w.sweep()
			require.Len(t, shares, 1, "once per pair, not once per account")
			assert.InDelta(t, tc.share, shares[0], 1e-9)

			shares = nil
			clock.Advance(2 * time.Minute)
			w.sweep()
			assert.Empty(t, shares, "nobody clicked since")
		})
	}
}
