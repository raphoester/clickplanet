package players

import "time"

// Day is one calendar day in UTC. The zero Day is no day.
type Day struct {
	midnight time.Time
}

// DayOf is the UTC day at falls on.
func DayOf(at time.Time) Day {
	year, month, day := at.UTC().Date()
	return Day{midnight: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

func (d Day) Empty() bool { return d.midnight.IsZero() }

// Following is the day after.
func (d Day) Following() Day { return Day{midnight: d.midnight.AddDate(0, 0, 1)} }

func (d Day) Before(other Day) bool { return d.midnight.Before(other.midnight) }

// String is YYYY-MM-DD, and empty for no day.
func (d Day) String() string {
	if d.Empty() {
		return ""
	}
	return d.midnight.Format(time.DateOnly)
}

// Stats is what a player did on the map. A player with no stats never took a tile.
type Stats struct {
	Account    AccountID
	TilesTaken uint64
	// StreakCurrent is the days in a row, ending on StreakLastDay, with at least one take.
	StreakCurrent uint32
	StreakBest    uint32
	StreakLastDay Day
}

// WithTake is the stats after one more tile taken at at.
//
// A take on the day after the last one extends the streak, a take on the same day changes nothing, and a
// gap starts it again at 1. A take older than the last day, which only a late event brings, counts as a
// tile and leaves the streak alone.
func (s Stats) WithTake(at time.Time) Stats {
	day := DayOf(at)
	s.TilesTaken++

	switch {
	case s.StreakLastDay.Empty():
		s.StreakCurrent = 1
	case day == s.StreakLastDay || day.Before(s.StreakLastDay):
		return s
	case day == s.StreakLastDay.Following():
		s.StreakCurrent++
	default:
		s.StreakCurrent = 1
	}

	s.StreakLastDay = day
	s.StreakBest = max(s.StreakBest, s.StreakCurrent)

	return s
}

// AsOf is the stats as a player reads them on today: a streak whose last day is before yesterday is over.
func (s Stats) AsOf(today Day) Stats {
	if s.StreakLastDay.Empty() || s.StreakLastDay == today || s.StreakLastDay.Following() == today {
		return s
	}
	s.StreakCurrent = 0
	return s
}
