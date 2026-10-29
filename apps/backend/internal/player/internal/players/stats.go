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
	missionsDone   uint64
	counted        Counted
}

// A fact can be told twice: one before its kind's mark was counted already. Ledger facts come in ledger order, then index order.
type Counted struct {
	TakesBelow     TakeMark
	BombsBelow     uint64
	FortifiesBelow uint64
	MessagesUntil  time.Time
	BonusesUntil   time.Time
	AnswersUntil   time.Time
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
	MissionsDone   uint64
	Counted        Counted
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
		missionsDone:   s.MissionsDone,
		counted:        s.Counted,
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
		MissionsDone:   s.missionsDone,
		Counted:        s.counted,
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

func (s Stats) MissionsDone() uint64 { return s.missionsDone }

type TakeMark struct {
	Position uint64
	Index    uint32
}

func takeMarkPast(position uint64, index uint32) TakeMark {
	return TakeMark{Position: position, Index: index + 1}
}

func (m TakeMark) counted(position uint64, index uint32) bool {
	return position < m.Position || position == m.Position && index < m.Index
}

func TakeAt(position uint64, index uint32, at time.Time) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithTake(position, index, at) }
}

func MessageAt(at time.Time) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithMessage(at) }
}

func BombAt(position uint64) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithBomb(position) }
}

func FortifyAt(position uint64, tiles uint64) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithFortify(position, tiles) }
}

func BonusAt(at time.Time) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithBonus(at) }
}

func RightAnswerAt(at time.Time) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithRightAnswer(at) }
}

func MissionsPaid(inAll uint64) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithMissionsPaid(inAll) }
}

// Postgres keeps a time to the microsecond: a fact told again must compare equal to the one kept.
func momentOf(at time.Time) time.Time { return at.UTC().Truncate(time.Microsecond) }

func (s Stats) WithTake(position uint64, index uint32, at time.Time) Stats {
	if s.counted.TakesBelow.counted(position, index) {
		return s
	}
	s.counted.TakesBelow = takeMarkPast(position, index)

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

func (s Stats) WithMessage(at time.Time) Stats {
	if !momentOf(at).After(s.counted.MessagesUntil) {
		return s
	}
	s.counted.MessagesUntil = momentOf(at)
	s.messagesSent++
	return s
}

func (s Stats) WithBomb(position uint64) Stats {
	if position < s.counted.BombsBelow {
		return s
	}
	s.counted.BombsBelow = position + 1
	s.bombsDropped++
	return s
}

func (s Stats) WithFortify(position uint64, tiles uint64) Stats {
	if position < s.counted.FortifiesBelow {
		return s
	}
	s.counted.FortifiesBelow = position + 1
	s.tilesFortified += tiles
	return s
}

func (s Stats) WithBonus(at time.Time) Stats {
	if !momentOf(at).After(s.counted.BonusesUntil) {
		return s
	}
	s.counted.BonusesUntil = momentOf(at)
	s.bonusesClaimed++
	return s
}

func (s Stats) WithRightAnswer(at time.Time) Stats {
	if !momentOf(at).After(s.counted.AnswersUntil) {
		return s
	}
	s.counted.AnswersUntil = momentOf(at)
	s.rightAnswers++
	return s
}

// Missions tells how many it paid in all, so a payment told twice or late counts nothing more.
func (s Stats) WithMissionsPaid(inAll uint64) Stats {
	s.missionsDone = max(s.missionsDone, inAll)
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
