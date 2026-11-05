package metronome

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsPayersThatStoppedTrying(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(Config{TrackWindow: time.Minute}, bounds(asIs), clock)

	w.Attempted(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: clock.Now()})
	require.Len(t, w.payers, 1)

	clock.Advance(2 * time.Hour)
	w.sweep()

	assert.Empty(t, w.payers)
}

func TestSweepReportsTheSkewOfAFullWindowOnly(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))

	var skews []float64
	ten := func() Rules { return bounds(func(r *Rules) { r.Shape.Clicks = 10 }) }
	w := New(Config{TrackWindow: time.Hour}, ten, clock, func(skew float64) { skews = append(skews, skew) }, func(float64) {}, func(time.Duration) {})

	gaps := []time.Duration{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for i, gap := range gaps {
		clock.Advance(gap * 100 * time.Millisecond)
		w.Attempted(detect.Click{Scope: "full", Tile: uint32(i), Country: "SZ", At: clock.Now()})
		w.Attempted(detect.Click{Scope: "short", Tile: uint32(i), Country: "SZ", At: clock.Now()})
	}
	clock.Advance(time.Second)
	w.Attempted(detect.Click{Scope: "full", Tile: 10, Country: "SZ", At: clock.Now()})

	w.sweep()

	require.Len(t, skews, 1, "the caller one gap short is not measured")
	assert.InDelta(t, 0, skews[0], 0.2)
}

func TestSweepReportsTheBusyTimeOfPayersStillClicking(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC))

	var busy []time.Duration
	config := Config{TrackWindow: time.Hour, Stamina: StaminaConfig{Slice: 10 * time.Minute}}
	hour := func() Rules { return bounds(func(r *Rules) { r.Stamina.Clicks, r.Stamina.Window = 2, time.Hour }) }
	w := New(config, hour, clock, func(float64) {}, func(float64) {}, func(b time.Duration) { busy = append(busy, b) })

	for range 30 {
		w.Watch(detect.Click{Scope: "still", Account: "a", At: clock.Now()})
		w.Watch(detect.Click{Scope: "gone", At: clock.Now().Add(-10 * time.Minute)})
		clock.Advance(time.Minute)
	}

	w.sweep()

	assert.Equal(t, []time.Duration{30 * time.Minute}, busy, "the payer that stopped before the sweep interval is not sampled")
}

func TestSweepForgetsPayersPastTheWindow(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(Config{TrackWindow: time.Minute}, bounds(func(r *Rules) { r.Stamina.Window = time.Hour }), clock)

	w.Watch(detect.Click{Scope: "caller", Account: "a", At: clock.Now()})
	clock.Advance(59 * time.Minute)
	w.sweep()
	require.Len(t, w.spenders, 1, "a slice inside the window is kept past trackWindow")

	clock.Advance(time.Hour)
	w.sweep()
	assert.Empty(t, w.spenders)
}

func TestTheShapeRingIsBounded(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(Config{TrackWindow: time.Hour}, bounds(func(r *Rules) { r.Shape.Clicks, r.Shape.CertainClicks = 8, 16 }), clock)

	for i := range uint32(500) {
		clock.Advance(time.Second)
		w.Attempted(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.Now()})
	}

	assert.Len(t, w.payers[detect.Click{Scope: "caller"}.Payer()].shape, 16)
}

func TestTheGapRingIsBounded(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	w := newWatchdog(Config{TrackWindow: time.Hour}, bounds(func(r *Rules) { r.MinClicks = 8 }), clock)

	for i := range uint32(500) {
		clock.Advance(time.Second)
		w.Attempted(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.Now()})
	}

	gaps := w.payers[detect.Click{Scope: "caller"}.Payer()].gaps
	require.NotEmpty(t, gaps)
	assert.LessOrEqual(t, len(gaps), 8,
		"only the gaps a spread needs are kept; how long the run has lasted is two timestamps")
}
