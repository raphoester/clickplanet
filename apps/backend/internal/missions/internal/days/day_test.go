package days_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
)

var starts = time.Date(2026, 10, 28, 21, 0, 0, 0, time.UTC)

func TestADayFollowsTheDayBeforeItInItsSeasonWhateverRanBetween(t *testing.T) {
	day := func(season, number uint32, starts time.Time) days.Day {
		day, err := days.DayOf(days.DaySnapshot{
			ID: days.DayID(fmt.Sprintf("season-%d/day-%d", season, number)), Book: days.BookID(fmt.Sprintf("season-%d", season)),
			Season: season, Number: number,
			StartsAt: starts, EndsAt: starts.Add(23 * time.Hour),
		})
		require.NoError(t, err)
		return day
	}
	sixth := day(1, 6, starts.Add(24*time.Hour))

	assert.True(t, sixth.Follows(day(1, 5, starts)), "the fifth ended an hour before the sixth: a battle ran")
	assert.False(t, sixth.Follows(day(1, 4, starts)))
	assert.False(t, sixth.Follows(day(0, 5, starts)))
	assert.False(t, day(1, 1, starts.Add(24*time.Hour)).Follows(day(0, 24, starts)), "a season starts with no day before")
}
