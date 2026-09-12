package retaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/detect"
)

type stubClock struct{ now time.Time }

func (c *stubClock) Now() time.Time { return c.now }

func TestSweepForgetsIdleCallersAndStaleTiles(t *testing.T) {
	clock := &stubClock{now: time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)}

	w := New(Config{ReactionWindow: time.Second, TrackWindow: time.Minute, MinReactions: 4}, clock, nil)

	player := detect.Click{Scope: "player", Tile: 1, Country: "FR", At: clock.now}
	w.Watch(player)
	w.Committed(player)

	clock.now = clock.now.Add(80 * time.Millisecond)

	bot := detect.Click{Scope: "bot", Tile: 1, Country: "PS", At: clock.now, Held: "FR"}
	w.Watch(bot)
	w.Committed(bot)

	require.Len(t, w.tiles, 1)
	require.Len(t, w.callers, 1)

	clock.now = clock.now.Add(2 * time.Hour)
	w.sweep()

	assert.Empty(t, w.tiles, "tiles nobody can still be reacting to are dropped")
	assert.Empty(t, w.callers, "callers with no reactions left are dropped")
}
