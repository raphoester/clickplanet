package hopper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsIdlePayers(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: 5 * time.Minute}, clock, func(float64) {})
	alternate(w, clock, 5)
	require.Len(t, w.payers, 1)

	clock.Advance(time.Hour)
	w.sweep()

	assert.Empty(t, w.payers)
}

func TestSweepKeepsAPayerThatCanStillMakeAStep(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))

	w := New(Config{}, clock, func(float64) {})
	w.Attempted(detect.Click{Scope: "2001:db8::/64", Account: "player", Position: north, At: clock.Now()})

	clock.Advance(10 * time.Second)
	w.sweep()

	assert.Len(t, w.payers, 1, "its next try, inside maxGap, is a step from this one")
}

func TestWithNoShareSetItOnlyMeasures(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC))

	var shares []float64
	w := New(Config{MinSteps: 10}, clock, func(share float64) { shares = append(shares, share) })

	for i := range 21 {
		clock.Advance(time.Second)
		at := north
		if i%2 == 1 {
			at = south
		}
		click := detect.Click{Scope: "2001:db8::/64", Account: "player", Position: at, At: clock.Now()}
		w.Attempted(click)

		verdict, _ := w.Watch(click)
		require.Equal(t, detect.Clear, verdict, "no bound set, nothing said")
	}

	w.sweep()

	assert.Equal(t, []float64{1}, shares)
}
