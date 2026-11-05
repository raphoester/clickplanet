package days_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
)

var (
	starts = time.Date(2026, 10, 28, 21, 0, 0, 0, time.UTC)
	ends   = starts.Add(24 * time.Hour)
)

func TestADayRunsFromItsStartToBeforeItsEnd(t *testing.T) {
	day, err := days.DayOf(days.DaySnapshot{ID: "season-0/day-22", StartsAt: starts, EndsAt: ends})
	require.NoError(t, err)

	assert.False(t, day.Runs(starts.Add(-time.Nanosecond)))
	assert.True(t, day.Runs(starts))
	assert.True(t, day.Runs(ends.Add(-time.Nanosecond)))
	assert.False(t, day.Runs(ends), "the next day starts at its end")
	assert.Equal(t, "season-0/day-22", day.String())
}

func TestADayHasAnIDAndEndsAfterItStarts(t *testing.T) {
	for _, snapshot := range []days.DaySnapshot{
		{StartsAt: starts, EndsAt: ends},
		{ID: "season-0/day-22", StartsAt: starts, EndsAt: starts},
		{ID: "season-0/day-22", StartsAt: ends, EndsAt: starts},
	} {
		_, err := days.DayOf(snapshot)
		assert.ErrorIs(t, err, days.ErrInvalidDay, "%+v", snapshot)
	}
}
