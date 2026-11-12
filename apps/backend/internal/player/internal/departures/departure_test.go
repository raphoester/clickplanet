package departures_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures"
)

var start = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

func TestAnAccountNobodyAskedToDeleteIsNotLeaving(t *testing.T) {
	assert.False(t, departures.Departure{}.TakeSnapshot().Leaving)
}

func TestADecisionHeardTwiceIsTheSame(t *testing.T) {
	left := departures.Departure{}.Decided(departures.Leaves(start))

	assert.Equal(t, left, left.Decided(departures.Leaves(start)))
	assert.Equal(t, left, left.Decided(departures.Returns(start)), "a restore at the same instant is not later")
}

func TestASnapshotRestoresTheSameDeparture(t *testing.T) {
	left := departures.Departure{}.Decided(departures.Leaves(start))

	assert.Equal(t, left, left.TakeSnapshot().Restore())
}
