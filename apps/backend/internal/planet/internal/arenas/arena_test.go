package arenas_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
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

func TestAnArenaOpensOnce(t *testing.T) {
	now := starts.Add(3 * time.Second)

	opening, opened, err := arenas.Arena{ID: "siege"}.Open(ends, now, arenas.Arena{})

	require.NoError(t, err)
	assert.True(t, opened)
	assert.Equal(t, arenas.Opening{ID: "siege", EndsAt: ends, OpenedAt: now}, opening)

	_, opened, err = arenas.Arena{ID: "siege", Opening: opening, Opened: true}.Open(ends, now.Add(time.Second), arenas.Arena{})
	require.NoError(t, err)
	assert.False(t, opened, "an arena open already changes nothing")
}

func TestAnArenaClosedEndedOrBesideAnotherIsRefused(t *testing.T) {
	now := starts.Add(3 * time.Second)

	_, _, closed := arenas.Arena{ID: "siege", Closed: true}.Open(ends, now, arenas.Arena{})
	_, _, ended := arenas.Arena{ID: "siege"}.Open(ends, ends, arenas.Arena{})
	_, _, beside := arenas.Arena{ID: "siege"}.Open(ends, now, arenas.Arena{ID: "other", Opened: true})

	require.ErrorIs(t, closed, arenas.ErrRefused)
	require.ErrorIs(t, ended, arenas.ErrRefused)
	require.ErrorIs(t, beside, arenas.ErrRefused)
}

func TestAnArenaClosesOnceOpenedOrNot(t *testing.T) {
	now := ends.Add(3 * time.Second)

	closing, closed := arenas.Arena{ID: "siege"}.Close(now)
	assert.True(t, closed, "an arena never opened closes too")
	assert.Equal(t, arenas.Closing{ID: "siege", ClosedAt: now}, closing)

	_, closed = arenas.Arena{ID: "siege", Closed: true}.Close(now)
	assert.False(t, closed)
}

func TestAnArenaFreezesAfterTheMapWentBlankAndBeforeTheBattleEnds(t *testing.T) {
	opening := arenas.Opening{ID: "siege", EndsAt: ends, OpenedAt: starts.Add(3 * time.Second)}
	freeze := 8 * time.Second

	assert.True(t, opening.Freezes(opening.OpenedAt, freeze))
	assert.True(t, opening.Freezes(opening.OpenedAt.Add(freeze-time.Millisecond), freeze))
	assert.False(t, opening.Freezes(opening.OpenedAt.Add(freeze), freeze))
	assert.False(t, opening.Freezes(ends.Add(-freeze-time.Millisecond), freeze))
	assert.True(t, opening.Freezes(ends.Add(-freeze), freeze))
	assert.True(t, opening.Freezes(ends.Add(time.Minute), freeze), "until the map is back")
	assert.False(t, opening.Freezes(starts.Add(time.Hour), 0))
}
