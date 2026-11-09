package arenas_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheArenaOpenAtBootFreezesAsTheRulesInForceSay(t *testing.T) {
	kept := arenas.NewMemoryStore()
	opening := arenas.Opening{ID: "siege", EndsAt: ends, OpenedAt: starts}
	require.NoError(t, kept.Open(t.Context(), opening, clicks.MapCopy{}))
	clock := cptime.NewFixedClock(starts.Add(5 * time.Second))
	freeze := 8 * time.Second
	inPlay := arenas.NewInPlay(func() arenas.Siege { return arenas.Siege{Freeze: freeze} }, clock)

	require.NoError(t, inPlay.Load(t.Context(), kept))

	assert.True(t, bool(inPlay.Freeze()))
	freeze = 3 * time.Second
	assert.False(t, bool(inPlay.Freeze()), "a set taken over changes the freeze at once")
}

func TestOnlyTheArenaOpenClosesIt(t *testing.T) {
	clock := cptime.NewFixedClock(starts)
	inPlay := arenas.NewInPlay(func() arenas.Siege { return arenas.Siege{Freeze: 8 * time.Second} }, clock)
	inPlay.Open(arenas.Opening{ID: "siege", EndsAt: ends, OpenedAt: starts})

	inPlay.Close("never-opened")
	assert.True(t, bool(inPlay.Freeze()))

	inPlay.Close("siege")
	assert.False(t, bool(inPlay.Freeze()))
}
