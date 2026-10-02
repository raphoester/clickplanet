package players

import "time"

type Day struct {
	midnight time.Time
}

func DayOf(at time.Time) Day {
	year, month, day := at.UTC().Date()
	return Day{midnight: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

func (d Day) Empty() bool { return d.midnight.IsZero() }

func (d Day) Following() Day { return Day{midnight: d.midnight.AddDate(0, 0, 1)} }

func (d Day) Before(other Day) bool { return d.midnight.Before(other.midnight) }

func (d Day) String() string {
	if d.Empty() {
		return ""
	}
	return d.midnight.Format(time.DateOnly)
}

type Stats struct {
	Account       AccountID
	TilesTaken    uint64
	StreakCurrent uint32
	StreakBest    uint32
	StreakLastDay Day
}

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

func (s Stats) AsOf(today Day) Stats {
	if s.StreakLastDay.Empty() || s.StreakLastDay == today || s.StreakLastDay.Following() == today {
		return s
	}
	s.StreakCurrent = 0
	return s
}
