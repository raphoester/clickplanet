package retaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestReactionsSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))
	config := Config{MinReactions: 5, TrackWindow: time.Hour}

	w := New(config, clock, nil)
	for tile := range uint32(6) {
		player := detect.Click{Scope: "player", Tile: tile, Country: "FR", At: clock.Now()}
		w.Watch(player)
		w.Committed(player)

		clock.Advance(100 * time.Millisecond)
		w.Watch(detect.Click{Scope: "reflex", Tile: tile, Country: "PS", At: clock.Now()})
		clock.Advance(time.Second)
	}

	data, err := w.Save()
	require.NoError(t, err)

	restarted := New(config, clock, nil)
	require.NoError(t, restarted.Load(data))

	next := detect.Click{Scope: "reflex", Tile: 99, Country: "PS", At: clock.Now(), NoOp: true}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Certain, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
	assert.Len(t, restarted.tiles, len(w.tiles))
}

func TestForgetDropsReactionsAndTakesBeforeTheCutoff(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{TrackWindow: 24 * time.Hour}, clock, nil)
	player := detect.Click{Scope: "player", Tile: 1, Country: "FR", At: clock.Now()}
	w.Watch(player)
	w.Committed(player)
	clock.Advance(time.Second)
	w.Watch(detect.Click{Scope: "reflex", Tile: 1, Country: "PS", At: clock.Now()})
	require.Len(t, w.callers, 1)

	w.Forget(clock.Now().Add(time.Second))

	assert.Empty(t, w.tiles)
	assert.Empty(t, w.callers)
}
