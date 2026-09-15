package defender

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTakesAndLossesSurviveASaveAndLoad(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))
	config := Config{MinClicks: 10, MinShare: 0.5, TrackWindow: time.Hour}

	w := New(config, clock, nil)
	for tile := range uint32(12) {
		attacker := detect.Click{Scope: "attacker", Tile: tile, Country: "FR", At: clock.Now(), Held: "BG"}
		w.Watch(attacker)
		w.Committed(attacker)

		clock.Advance(10 * time.Second)
		bot := detect.Click{Scope: "bot", Tile: tile, Country: "BG", At: clock.Now(), Held: "FR"}
		w.Watch(bot)
		w.Committed(bot)
	}

	// A loss taken just before the restart, retaken just after.
	attacker := detect.Click{Scope: "attacker", Tile: 500, Country: "FR", At: clock.Now(), Held: "BG"}
	w.Watch(attacker)
	w.Committed(attacker)

	data, err := w.Save()
	require.NoError(t, err)

	restarted := New(config, clock, nil)
	require.NoError(t, restarted.Load(data))

	clock.Advance(20 * time.Second)
	next := detect.Click{Scope: "bot", Tile: 500, Country: "BG", At: clock.Now(), Held: "FR"}
	wantVerdict, wantEvidence := w.Watch(next)
	gotVerdict, gotEvidence := restarted.Watch(next)

	require.Equal(t, detect.Suspect, wantVerdict)
	assert.Equal(t, wantVerdict, gotVerdict)
	assert.Equal(t, wantEvidence, gotEvidence)
}

func TestForgetDropsTakesAndLossesBeforeTheCutoff(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC))

	w := New(Config{RetakeWindow: time.Hour, TrackWindow: 24 * time.Hour}, clock, nil)
	attacker := detect.Click{Scope: "attacker", Tile: 1, Country: "FR", At: clock.Now(), Held: "BG"}
	w.Watch(attacker)
	w.Committed(attacker)

	w.Forget(clock.Now().Add(time.Second))

	assert.Empty(t, w.losses)
	assert.Empty(t, w.callers)
}
