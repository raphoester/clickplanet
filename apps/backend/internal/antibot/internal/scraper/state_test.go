package scraper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestReadsSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC))
	config := Config{MinMaps: 5, CertainMaps: 15, TrackWindow: 15 * time.Minute}

	w := New(config, clock, func(float64) {})
	w.Listened("bot")
	for range 500 {
		clock.Advance(time.Second)
		w.Fetched("bot", 0.04)
	}
	w.Attempted(detect.Click{Scope: "bot", Tile: 1, Country: "dz", At: clock.Now()})

	data, err := w.Save()
	require.NoError(t, err)

	restarted := New(config, clock, func(float64) {})
	require.NoError(t, restarted.Load(data))

	next := detect.Click{Scope: "bot", Tile: 1, Country: "dz", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Certain, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
	assert.True(t, restarted.callers["bot"].clicked.Equal(clock.Now()), "still a caller that clicks, for the histogram")
}

func TestForgetDropsSlicesBeforeTheCutoff(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 15, 21, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: 24 * time.Hour}, clock, func(float64) {})
	w.Fetched("player", 1)
	clock.Advance(time.Hour)
	w.Fetched("player", 1)

	w.Forget(clock.Now().Add(-time.Second))
	require.Len(t, w.callers["player"].slices, 1)

	w.Forget(clock.Now().Add(time.Second))
	assert.Empty(t, w.callers)
}
