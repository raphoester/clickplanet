package catcher

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func bounds(change func(*Rules)) func() Rules {
	r := Rules{
		MinCatches:    5,
		MaxMedian:     3 * time.Second,
		CertainMedian: 1500 * time.Millisecond,
		Foreign:       ForeignRules{Window: time.Hour},
	}
	change(&r)
	return func() Rules { return r }
}

func asIs(*Rules) {}

func TestCatchesSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))
	config := Config{TrackWindow: time.Hour}

	w := New(config, bounds(asIs), clock)
	for range 5 {
		clock.Advance(30 * time.Second)
		w.Caught("snatcher", 400*time.Millisecond)
	}

	data, err := w.Save()
	require.NoError(t, err)

	restarted := New(config, bounds(asIs), clock)
	require.NoError(t, restarted.Load(data))

	next := detect.Click{Scope: "snatcher", Tile: 1, Country: "FR", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Certain, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
}

func TestForgetDropsBoxesBeforeTheCutoff(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: 24 * time.Hour}, bounds(asIs), clock)
	w.Missed("player")
	clock.Advance(time.Minute)
	w.Caught("player", time.Second)

	w.Forget(clock.Now().Add(-time.Second))
	require.Len(t, w.callers["player"].outcomes, 1)

	w.Forget(clock.Now().Add(time.Second))
	assert.Empty(t, w.callers)
}
