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

	w := New(Config{TrackWindow: time.Minute}, clock)

	w.Watch(detect.Click{Scope: "caller", Tile: 1, Country: "FR", At: clock.Now()})
	require.Len(t, w.callers, 1)

	clock.Advance(2 * time.Hour)
	w.sweep()

	assert.Empty(t, w.callers)
}

func TestTheGapRingIsBounded(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	w := New(Config{MaxGap: 3 * time.Second, MinClicks: 8, TrackWindow: time.Hour}, clock)

	for i := range uint32(500) {
		clock.Advance(time.Second)
		w.Watch(detect.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.Now()})
	}

	assert.LessOrEqual(t, len(w.callers["caller"].gaps), 8,
		"only the gaps a spread needs are kept; how long the run has lasted is two timestamps")
}
