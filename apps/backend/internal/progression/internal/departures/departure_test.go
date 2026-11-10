package departures_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures"
)

var start = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)

func TestAnAccountNobodyAskedToDeleteIsNotLeaving(t *testing.T) {
	assert.False(t, departures.Departure{}.TakeSnapshot().Leaving)
}

func TestARequestMakesTheAccountLeaveAndARestoreBringsItBack(t *testing.T) {
	left := departures.Departure{}.Decided(departures.Leaves(start))
	assert.True(t, left.TakeSnapshot().Leaving)

	back := left.Decided(departures.Returns(start.Add(time.Hour)))
	assert.False(t, back.TakeSnapshot().Leaving)
}

func TestAnOlderDecisionHeardLateChangesNothing(t *testing.T) {
	back := departures.Departure{}.Decided(departures.Returns(start.Add(time.Hour)))

	assert.False(t, back.Decided(departures.Leaves(start)).TakeSnapshot().Leaving, "the request was before the restore")
}

func TestADecisionHeardTwiceIsTheSame(t *testing.T) {
	left := departures.Departure{}.Decided(departures.Leaves(start))

	assert.Equal(t, left, left.Decided(departures.Leaves(start)))
	assert.Equal(t, left, left.Decided(departures.Returns(start)), "a restore at the same instant is not later")
}

func TestADecisionWithNoTimeIsAnError(t *testing.T) {
	assert.ErrorIs(t, departures.Leaves(time.Time{}).TimeError(), departures.ErrNoTime)
	assert.NoError(t, departures.Returns(start).TimeError())
}

func TestASnapshotRestoresTheSameDeparture(t *testing.T) {
	left := departures.Departure{}.Decided(departures.Leaves(start))

	assert.Equal(t, left, left.TakeSnapshot().Restore())
}
