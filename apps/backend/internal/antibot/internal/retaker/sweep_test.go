package retaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestSweepForgetsIdleCallersAndStaleTiles(t *testing.T) {
	clock := cptime.NewFixedClock(time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC))

	rules := bounds(func(r *Rules) {
		r.ReactionWindow = time.Second
		r.MinReactions = 4
	})
	w := New(Config{TrackWindow: time.Minute}, rules, clock, nil)
	empty, err := New(Config{TrackWindow: time.Minute}, rules, clock, nil).Save()
	require.NoError(t, err)

	player := detect.Click{Scope: "player", Tile: 1, Country: "FR", At: clock.Now()}
	w.Watch(player)
	w.Committed(player)

	clock.Advance(80 * time.Millisecond)

	bot := detect.Click{Scope: "bot", Tile: 1, Country: "PS", At: clock.Now(), Held: "FR"}
	w.Watch(bot)
	w.Committed(bot)

	watched, err := w.Save()
	require.NoError(t, err)
	require.NotEqual(t, empty, watched)

	clock.Advance(2 * time.Hour)
	w.sweep()

	swept, err := w.Save()
	require.NoError(t, err)
	assert.Equal(t, empty, swept, "tiles nobody can still be reacting to and callers with no reactions left are dropped")
}
