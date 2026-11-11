package arenas_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
)

var (
	openedAt = time.Date(2026, 11, 11, 21, 0, 3, 400_000_000, time.UTC)
	closedAt = time.Date(2026, 11, 11, 22, 0, 2, 700_000_000, time.UTC)
)

func closedStretch() arenas.Stretch {
	return arenas.Stretch{ID: "battle", OpenedAt: openedAt, ClosedAt: closedAt, Closed: true}
}

func TestAStretchHoldsWhatTheLedgerKeptAfterItsOpenAndBeforeItsCloseSecond(t *testing.T) {
	stretch := closedStretch()

	assert.False(t, stretch.Holds(openedAt.Truncate(time.Second)), "a world take of the open's second, before it")
	assert.True(t, stretch.Holds(openedAt.Truncate(time.Second).Add(time.Second)), "the second after the open")
	assert.True(t, stretch.Holds(closedAt.Truncate(time.Second).Add(-time.Second)), "the second before the close's")
	assert.False(t, stretch.Holds(closedAt.Truncate(time.Second)), "a world take right after the close shares its second")
	assert.False(t, stretch.Holds(closedAt.Add(time.Hour)))
}

func TestAStretchStillOpenHoldsEverythingAfterItsOpen(t *testing.T) {
	stretch := arenas.Stretch{ID: "battle", OpenedAt: openedAt}

	assert.False(t, stretch.Holds(openedAt.Add(-time.Second)))
	assert.True(t, stretch.Holds(openedAt.Add(24*time.Hour)))
	until := closedAt.Add(time.Minute)
	assert.Equal(t, until, stretch.Before(until), "an open stretch ends at the window's end")
	assert.Equal(t, closedAt.Truncate(time.Second), closedStretch().Before(until))
}

func TestTheWorldPlayedAroundEveryStretch(t *testing.T) {
	world := arenas.MapsOf([]arenas.Stretch{closedStretch()}).World()

	assert.True(t, world.Holds(openedAt.Add(-time.Minute)))
	assert.False(t, world.Holds(openedAt.Add(time.Minute)))
	assert.True(t, world.Holds(closedAt))
}

func TestTheMapThatPlaysIsTheOpenArenaSinceItOpenedOrTheWorld(t *testing.T) {
	later := openedAt.Add(2 * time.Hour)
	open := arenas.Stretch{ID: "next", OpenedAt: later}
	maps := arenas.MapsOf([]arenas.Stretch{closedStretch(), open})

	playing := maps.Playing()
	assert.False(t, playing.Holds(closedAt.Add(time.Minute)), "a world take before the open")
	assert.True(t, playing.Holds(later.Add(time.Minute)))
	stretch, ok := maps.Open()
	assert.True(t, ok)
	assert.Equal(t, open, stretch)

	world := arenas.MapsOf([]arenas.Stretch{closedStretch()})
	_, ok = world.Open()
	assert.False(t, ok)
	assert.True(t, world.Playing().Holds(closedAt.Add(time.Minute)))
	assert.False(t, world.Playing().Holds(openedAt.Add(time.Minute)))
}

func TestTheAsidesAreTheStretchesThatPlayedSomeOfAWindow(t *testing.T) {
	maps := arenas.MapsOf([]arenas.Stretch{closedStretch()})

	assert.Len(t, maps.Asides(openedAt.Add(-time.Hour), openedAt.Add(time.Second)), 1)
	assert.Len(t, maps.Asides(openedAt.Add(time.Minute), openedAt.Add(2*time.Minute)), 1, "a window inside it")
	assert.Empty(t, maps.Asides(closedAt, closedAt.Add(time.Hour)))
	assert.Empty(t, maps.Asides(openedAt.Add(-time.Hour), openedAt))
}
