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

	w := New(Config{TrackWindow: time.Minute}, bounds(asIs), clock)
	empty, err := New(Config{TrackWindow: time.Minute}, bounds(asIs), clock).Save()
	require.NoError(t, err)

	w.Caught("caller", time.Second)
	watched, err := w.Save()
	require.NoError(t, err)
	require.NotEqual(t, empty, watched)

	clock.Advance(2 * time.Hour)
	w.sweep()

	swept, err := w.Save()
	require.NoError(t, err)
	assert.Equal(t, empty, swept)
}

func TestOnlyTheLastBoxesAreKept(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{}, bounds(asIs), clock)

	for range 100 {
		clock.Advance(time.Minute)
		w.Missed("caller")
	}

	assert.Len(t, w.callers["caller"].outcomes, 5)
}

func TestSweepKeepsACallerThatOnlyClaimedOtherBoxes(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: time.Minute}, bounds(func(r *Rules) { r.Foreign.MinClaims = 3 }), clock)

	w.Foreign("caller")
	clock.Advance(30 * time.Minute)
	kept, err := w.Save()
	require.NoError(t, err)
	w.sweep()

	swept, err := w.Save()
	require.NoError(t, err)
	require.Equal(t, kept, swept, "inside the foreign window")

	clock.Advance(time.Hour)
	w.sweep()
	swept, err = w.Save()
	require.NoError(t, err)
	empty, err := New(Config{TrackWindow: time.Minute}, bounds(asIs), clock).Save()
	require.NoError(t, err)
	assert.Equal(t, empty, swept)
}
