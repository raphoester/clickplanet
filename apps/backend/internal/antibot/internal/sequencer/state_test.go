package sequencer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestStepsSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))
	config := Config{MinSteps: 20, CertainSteps: 40, TrackWindow: time.Hour}

	w := New(config, clock)
	for tile := range uint32(30) {
		clock.Advance(time.Second)
		w.Watch(detect.Click{Scope: "sweeper", Tile: 1000 + tile, Country: "FR", At: clock.Now()})
	}

	data, err := w.Save()
	require.NoError(t, err)

	restarted := New(config, clock)
	require.NoError(t, restarted.Load(data))

	// The next id continues the stride only if the last tile came back too.
	clock.Advance(time.Second)
	next := detect.Click{Scope: "sweeper", Tile: 1030, Country: "FR", At: clock.Now()}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Suspect, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
}

func TestForgetDropsStepsBeforeTheCutoff(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: 24 * time.Hour}, clock)
	for tile := range uint32(3) {
		clock.Advance(time.Second)
		w.Watch(detect.Click{Scope: "sweeper", Tile: tile, Country: "FR", At: clock.Now()})
	}

	w.Forget(clock.Now().Add(-time.Second))
	require.Len(t, w.callers["sweeper"].steps, 1)

	w.Forget(clock.Now().Add(time.Second))
	assert.Empty(t, w.callers)
}
