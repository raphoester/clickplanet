package arenas_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheArenaOpenAtBootFreezesAsItsOpeningSays(t *testing.T) {
	opening := arenas.Opening{ID: "battle", EndsAt: ends, OpenedAt: starts, Freeze: 8 * time.Second}
	clock := cptime.NewFixedClock(starts.Add(5 * time.Second))
	inPlay := arenas.NewInPlay(clock)

	inPlay.Resume(arenas.Arena{ID: "battle", Opening: opening, Opened: true})

	assert.True(t, bool(inPlay.Freeze()))
	clock.Advance(3 * time.Second)
	assert.False(t, bool(inPlay.Freeze()))
}

func TestAnArenaClosedAtBootIsNotResumed(t *testing.T) {
	clock := cptime.NewFixedClock(starts)
	inPlay := arenas.NewInPlay(clock)

	inPlay.Resume(arenas.Arena{ID: "battle", Opening: arenas.Opening{ID: "battle", EndsAt: ends, OpenedAt: starts, Freeze: 8 * time.Second}, Opened: true, Closed: true})

	assert.False(t, bool(inPlay.Freeze()))
}

func TestOnlyTheArenaOpenClosesIt(t *testing.T) {
	clock := cptime.NewFixedClock(starts)
	inPlay := arenas.NewInPlay(clock)
	inPlay.Open(arenas.Opening{ID: "battle", EndsAt: ends, OpenedAt: starts, Freeze: 8 * time.Second})

	inPlay.Close("never-opened")
	assert.True(t, bool(inPlay.Freeze()))

	inPlay.Close("battle")
	assert.False(t, bool(inPlay.Freeze()))
}
