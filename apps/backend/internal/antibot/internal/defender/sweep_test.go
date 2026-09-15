package defender

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsIdleCallersAndOldLosses(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 18, 0, 0, 0, time.UTC))

	w := New(Config{RetakeWindow: time.Minute, TrackWindow: 5 * time.Minute}, clock, nil)

	attacker := detect.Click{Scope: "attacker", Tile: 1, Country: "FR", At: clock.Now(), Held: "BG"}
	w.Watch(attacker)
	w.Committed(attacker)

	require.Len(t, w.losses, 1)
	require.Len(t, w.callers, 1)

	clock.Advance(time.Hour)
	w.sweep()

	assert.Empty(t, w.losses)
	assert.Empty(t, w.callers)
}

func TestWithNoShareSetItOnlyMeasures(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 14, 18, 0, 0, 0, time.UTC))

	var shares []float64
	w := New(Config{MinClicks: 10}, clock, func(share float64) { shares = append(shares, share) })

	for tile := range uint32(40) {
		attacker := detect.Click{Scope: "attacker", Tile: tile, Country: "FR", At: clock.Now(), Held: "BG"}
		w.Watch(attacker)
		w.Committed(attacker)

		clock.Advance(time.Second)

		bot := detect.Click{Scope: "bot", Tile: tile, Country: "BG", At: clock.Now(), Held: "FR"}
		verdict, _ := w.Watch(bot)
		require.Equal(t, detect.Clear, verdict, "no bound set, nothing said")
		w.Committed(bot)
	}

	w.sweep()

	require.Len(t, shares, 2, "both callers are past MinClicks")
	assert.ElementsMatch(t, []float64{0, 1}, shares)
}
