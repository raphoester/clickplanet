package metronome

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsIdleCallers(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: time.Minute}, clock, func(float64) {}, func(time.Duration) {})

	w.Attempted(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: clock.Now()})
	require.Len(t, w.callers, 1)

	clock.Advance(2 * time.Hour)
	w.sweep()

	assert.Empty(t, w.callers)
}

func TestSweepReportsTheSkewOfAFullWindowOnly(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC))

	var skews []float64
	w := New(Config{TrackWindow: time.Hour, Shape: ShapeConfig{Clicks: 10}}, clock, func(skew float64) { skews = append(skews, skew) }, func(time.Duration) {})

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
	config := Config{TrackWindow: time.Hour, Stamina: StaminaConfig{Slice: 10 * time.Minute, Clicks: 2, Window: time.Hour}}
	w := New(config, clock, func(float64) {}, func(b time.Duration) { busy = append(busy, b) })

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

	w := New(Config{TrackWindow: time.Minute, Stamina: StaminaConfig{Window: time.Hour}}, clock, func(float64) {}, func(time.Duration) {})

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

	w := New(Config{TrackWindow: time.Hour, Shape: ShapeConfig{Clicks: 8, CertainClicks: 16}}, clock, func(float64) {}, func(time.Duration) {})

	for i := range uint32(500) {
		clock.Advance(time.Second)
		w.Attempted(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.Now()})
	}

	assert.Len(t, w.callers["caller"].shape, 16)
}

func TestTheGapRingIsBounded(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	w := New(Config{MaxGap: 3 * time.Second, MinClicks: 8, TrackWindow: time.Hour}, clock, func(float64) {}, func(time.Duration) {})

	for i := range uint32(500) {
		clock.Advance(time.Second)
		w.Attempted(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.Now()})
	}

	assert.LessOrEqual(t, len(w.callers["caller"].gaps), 8,
		"only the gaps a spread needs are kept; how long the run has lasted is two timestamps")
}
