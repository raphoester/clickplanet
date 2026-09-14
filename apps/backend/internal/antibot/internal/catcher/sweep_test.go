package catcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsIdleCallers(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: time.Minute}, clock)

	w.Caught("caller", time.Second)
	require.Len(t, w.callers, 1)

	clock.Advance(2 * time.Hour)
	w.sweep()

	assert.Empty(t, w.callers)
}

func TestOnlyTheLastBoxesAreKept(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{MinCatches: 5}, clock)

	for range 100 {
		clock.Advance(time.Minute)
		w.Missed("caller")
	}

	assert.Len(t, w.callers["caller"].outcomes, 5)
}
