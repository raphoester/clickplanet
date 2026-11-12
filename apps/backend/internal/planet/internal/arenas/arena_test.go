package arenas_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var (
	starts = time.Date(2026, 11, 10, 21, 0, 0, 0, time.UTC)
	ends   = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)
)

func TestAnIDIsOneTo64Characters(t *testing.T) {
	_, err := arenas.ParseID("")
	require.ErrorIs(t, err, arenas.ErrInvalidArena)
	_, err = arenas.ParseID(string(make([]rune, 65)))
	require.ErrorIs(t, err, arenas.ErrInvalidArena)
	id, err := arenas.ParseID("season-1/finale")
	require.NoError(t, err)
	assert.Equal(t, arenas.ID("season-1/finale"), id)
}

func TestTheWorldIsNoArena(t *testing.T) {
	_, err := arenas.ParseID("world")
	require.ErrorIs(t, err, arenas.ErrInvalidArena)
}

func TestTheMapThatPlaysIsTheArenasWhileItIsOpen(t *testing.T) {
	assert.Equal(t, clicks.World, arenas.Arena{}.Played())
	assert.Equal(t, clicks.MapID("battle"), arenas.Arena{ID: "battle", Opened: true}.Played())
	assert.Equal(t, clicks.World, arenas.Arena{ID: "battle", Opened: true, Closed: true}.Played())
}

func TestAnArenaOpensOnce(t *testing.T) {
	now := starts.Add(3 * time.Second)

	opening, opened, err := arenas.Arena{ID: "battle"}.Open(ends, now, 8*time.Second, arenas.Arena{})

	require.NoError(t, err)
	assert.True(t, opened)
	assert.Equal(t, arenas.Opening{ID: "battle", EndsAt: ends, OpenedAt: now, Freeze: 8 * time.Second}, opening)

	_, opened, err = arenas.Arena{ID: "battle", Opening: opening, Opened: true}.Open(ends, now.Add(time.Second), 8*time.Second, arenas.Arena{})
	require.NoError(t, err)
	assert.False(t, opened, "an arena open already changes nothing")
}

func TestAnArenaFreezesAfterItsMapPlaysAndBeforeTheBattleEnds(t *testing.T) {
	freeze := 8 * time.Second
	opening := arenas.Opening{ID: "battle", EndsAt: ends, OpenedAt: starts.Add(3 * time.Second), Freeze: freeze}

	assert.True(t, opening.Freezes(opening.OpenedAt))
	assert.True(t, opening.Freezes(opening.OpenedAt.Add(freeze-time.Millisecond)))
	assert.False(t, opening.Freezes(opening.OpenedAt.Add(freeze)))
	assert.False(t, opening.Freezes(ends.Add(-freeze-time.Millisecond)))
	assert.True(t, opening.Freezes(ends.Add(-freeze)))
	assert.True(t, opening.Freezes(ends.Add(time.Minute)), "until the map is back")
	opening.Freeze = 0
	assert.False(t, opening.Freezes(starts.Add(time.Hour)))
}
