//go:build testing

package days

import (
	"fmt"
	"time"
)

var seasonZero = time.Date(2026, 10, 7, 21, 0, 0, 0, time.UTC)

// TestingDayAt is the day of season 0 that runs at a time, as the gamemaster tells it: day K ends at 21:00 UTC, K days
// after 7 Oct 2026.
func TestingDayAt(at time.Time) Day {
	number := int(at.Sub(seasonZero)/(24*time.Hour)) + 1
	ends := seasonZero.Add(time.Duration(number) * 24 * time.Hour)
	day, err := DayOf(DaySnapshot{ID: DayID(fmt.Sprintf("season-0/day-%d", number)), StartsAt: ends.Add(-24 * time.Hour), EndsAt: ends})
	if err != nil {
		panic(err)
	}
	return day
}
