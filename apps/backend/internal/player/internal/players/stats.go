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
	account        AccountID
	tilesTaken     uint64
	streak         Streak
	streakBest     uint32
	messagesSent   uint64
	bombsDropped   uint64
	tilesFortified uint64
	bonusesClaimed uint64
	rightAnswers   uint64
}

type StatsSnapshot struct {
	Account        AccountID
	TilesTaken     uint64
	Streak         Streak
	StreakBest     uint32
	MessagesSent   uint64
	BombsDropped   uint64
	TilesFortified uint64
	BonusesClaimed uint64
	RightAnswers   uint64
}

func (s StatsSnapshot) Restore() Stats {
	return Stats{
		account:        s.Account,
		tilesTaken:     s.TilesTaken,
		streak:         s.Streak,
		streakBest:     s.StreakBest,
		messagesSent:   s.MessagesSent,
		bombsDropped:   s.BombsDropped,
		tilesFortified: s.TilesFortified,
		bonusesClaimed: s.BonusesClaimed,
		rightAnswers:   s.RightAnswers,
	}
}

func (s Stats) TakeSnapshot() StatsSnapshot {
	return StatsSnapshot{
		Account:        s.account,
		TilesTaken:     s.tilesTaken,
		Streak:         s.streak,
		StreakBest:     s.streakBest,
		MessagesSent:   s.messagesSent,
		BombsDropped:   s.bombsDropped,
		TilesFortified: s.tilesFortified,
		BonusesClaimed: s.bonusesClaimed,
		RightAnswers:   s.rightAnswers,
	}
}

func NewStats(account AccountID) Stats {
	return Stats{account: account}
}

func AccountsOf(page []Stats) []AccountID {
	accounts := make([]AccountID, 0, len(page))
	for _, stats := range page {
		accounts = append(accounts, stats.Account())
	}
	return accounts
}

func (s Stats) Account() AccountID { return s.account }

func (s Stats) TilesTaken() uint64 { return s.tilesTaken }

func (s Stats) Streak() Streak { return s.streak }

func (s Stats) StreakBest() uint32 { return s.streakBest }

func (s Stats) MessagesSent() uint64 { return s.messagesSent }

func (s Stats) BombsDropped() uint64 { return s.bombsDropped }

func (s Stats) TilesFortified() uint64 { return s.tilesFortified }

func (s Stats) BonusesClaimed() uint64 { return s.bonusesClaimed }

func (s Stats) RightAnswers() uint64 { return s.rightAnswers }

func TakeAt(at time.Time) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithTake(at) }
}

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

func (s Stats) WithBomb() Stats {
	s.bombsDropped++
	return s
}

func (s Stats) WithFortify(tiles uint64) Stats {
	s.tilesFortified += tiles
	return s
}

func (s Stats) WithBonus() Stats {
	s.bonusesClaimed++
	return s
}

func (s Stats) WithRightAnswer() Stats {
	s.rightAnswers++
	return s
}

type Recount struct {
	BombsDropped   uint64
	TilesFortified uint64
}

func (s Stats) Recounted(recount Recount) Stats {
	s.bombsDropped, s.tilesFortified = recount.BombsDropped, recount.TilesFortified
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
