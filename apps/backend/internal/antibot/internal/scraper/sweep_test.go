package scraper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsIdleCallers(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: time.Minute}, clock, func(float64) {})

	w.Listened("player")
	w.Fetched("player", 1)
	require.Len(t, w.callers, 1)

	clock.Advance(2 * time.Minute)
	w.sweep()

	assert.Empty(t, w.callers)
}

func TestAFloodOfReadsKeepsOneSliceAStep(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: 15 * time.Minute}, clock, func(float64) {})

	for range 100_000 {
		clock.Advance(20 * time.Millisecond)
		w.Fetched("flooder", 0.001)
	}

	assert.LessOrEqual(t, len(w.callers["flooder"].slices), steps+1, "GetMap is not throttled: memory must not follow the pace")
}

func TestSweepReportsTheCallersThatClick(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC))

	var reported []float64
	w := New(Config{TrackWindow: 15 * time.Minute}, clock, func(maps float64) { reported = append(reported, maps) })

	w.Fetched("viewer", 3)

	w.Listened("clicker")
	w.Fetched("clicker", 8)
	w.Attempted(detect.Click{Scope: "clicker", Tile: 1, Country: "dz", At: clock.Now()})

	w.Listened("reconnecting")
	w.Listened("reconnecting")
	w.Fetched("reconnecting", 1)
	w.Attempted(detect.Click{Scope: "reconnecting", Tile: 2, Country: "fr", At: clock.Now()})

	w.sweep()

	assert.ElementsMatch(t, []float64{7, 0}, reported, "the viewer never clicked, and a reconnect never counts below zero")
}
