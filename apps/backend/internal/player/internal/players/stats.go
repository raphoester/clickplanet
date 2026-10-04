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
	account      AccountID
	tilesTaken   uint64
	streak       Streak
	streakBest   uint32
	messagesSent uint64
}

func NewStats(account AccountID) Stats {
	return Stats{account: account}
}

func StatsOf(account AccountID, tilesTaken uint64, streak Streak, streakBest uint32, messagesSent uint64) Stats {
	return Stats{account: account, tilesTaken: tilesTaken, streak: streak, streakBest: streakBest, messagesSent: messagesSent}
}

func (s Stats) Account() AccountID { return s.account }

func (s Stats) TilesTaken() uint64 { return s.tilesTaken }

func (s Stats) Streak() Streak { return s.streak }

func (s Stats) StreakBest() uint32 { return s.streakBest }

func (s Stats) MessagesSent() uint64 { return s.messagesSent }

func (s Stats) WithTake(at time.Time) Stats {
	day := DayOf(at)
	s.tilesTaken++

	switch {
	case s.streak.lastDay.Empty():
		s.streak.days = 1
	case day == s.streak.lastDay || day.Before(s.streak.lastDay):
		return s
	case day == s.streak.lastDay.Following():
		s.streak.days++
	default:
		s.streak.days = 1
	}

	s.streak.lastDay = day
	s.streakBest = max(s.streakBest, s.streak.days)

	return s
}

func (s Stats) WithMessage() Stats {
	s.messagesSent++
	return s
}

func (s Stats) AsOf(today Day) Stats {
	s.streak = s.streak.AsOf(today)
	return s
}

type Streak struct {
	days    uint32
	lastDay Day
}

func StreakOf(days uint32, lastDay Day) Streak {
	return Streak{days: days, lastDay: lastDay}
}

func (s Streak) Days() uint32 { return s.days }

func (s Streak) LastDay() Day { return s.lastDay }

func (s Streak) AsOf(today Day) Streak {
	if s.lastDay.Empty() || s.lastDay == today || s.lastDay.Following() == today {
		return s
	}
	s.days = 0
	return s
}
