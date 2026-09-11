package sequencer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
)

type stubClock struct{ now time.Time }

func (c *stubClock) Now() time.Time { return c.now }

func TestSweepForgetsIdleCallers(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}

	w := New(Config{TrackWindow: time.Minute}, clock)

	for i := range uint32(5) {
		w.Watch(antibot.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.now})
	}
	require.Len(t, w.callers, 1)

	clock.now = clock.now.Add(2 * time.Hour)
	w.sweep()

	assert.Empty(t, w.callers)
}

func TestTheStepRingIsBounded(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}

	w := New(Config{MinSteps: 4, CertainSteps: 16, TrackWindow: time.Hour}, clock)

	for i := range uint32(500) {
		w.Watch(antibot.Click{Scope: "caller", Tile: i, Country: "FR", At: clock.now})
	}

	assert.LessOrEqual(t, len(w.callers["caller"].steps), 16,
		"a caller clicking for hours must not cost memory for hours")
}
