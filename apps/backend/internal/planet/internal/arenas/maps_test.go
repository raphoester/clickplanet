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

func TestTheAsidesAreTheStretchesThatPlayedSomeOfAWindow(t *testing.T) {
	maps := arenas.MapsOf([]arenas.Stretch{closedStretch()})

	assert.Len(t, maps.Asides(openedAt.Add(-time.Hour), openedAt.Add(time.Second)), 1)
	assert.Len(t, maps.Asides(openedAt.Add(time.Minute), openedAt.Add(2*time.Minute)), 1, "a window inside it")
	assert.Empty(t, maps.Asides(closedAt, closedAt.Add(time.Hour)))
	assert.Empty(t, maps.Asides(openedAt.Add(-time.Hour), openedAt))
}
