package sequencer

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

	w := New(Config{TrackWindow: time.Minute}, bounds(asIs), clock)
	empty, err := New(Config{TrackWindow: time.Minute}, bounds(asIs), clock).Save()
	require.NoError(t, err)

	for i := range uint32(5) {
		w.Watch(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.Now()})
	}
	watched, err := w.Save()
	require.NoError(t, err)
	require.NotEqual(t, empty, watched)

	clock.Advance(2 * time.Hour)
	w.sweep()

	swept, err := w.Save()
	require.NoError(t, err)
	assert.Equal(t, empty, swept)
}

func TestTheStepRingIsBounded(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: time.Hour}, bounds(func(r *Rules) { r.MinSteps, r.CertainSteps = 4, 16 }), clock)

	for i := range uint32(500) {
		w.Watch(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.Now()})
	}

	assert.LessOrEqual(t, len(w.callers["caller"].steps), 16,
		"a caller clicking for hours must not cost memory for hours")
}
