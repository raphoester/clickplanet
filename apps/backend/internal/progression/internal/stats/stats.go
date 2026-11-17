package stats

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Day struct {
	midnight time.Time
}

func DayOf(at time.Time) Day {
	year, month, day := at.UTC().Date()
	return Day{midnight: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// DayFrom reads a day written as YYYY-MM-DD, as Day.String writes it.
func DayFrom(value string) (Day, error) {
	at, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return Day{}, fmt.Errorf("not a day: %q: %w", value, err)
	}
	return DayOf(at), nil
}

func (d Day) Empty() bool { return d.midnight.IsZero() }

func (d Day) Following() Day { return Day{midnight: d.midnight.AddDate(0, 0, 1)} }

func (d Day) Preceding() Day { return Day{midnight: d.midnight.AddDate(0, 0, -1)} }

func (d Day) Before(other Day) bool { return d.midnight.Before(other.midnight) }

func (d Day) Compare(other Day) int { return d.midnight.Compare(other.midnight) }

func (d Day) String() string {
	if d.Empty() {
		return ""
	}
	return d.midnight.Format(time.DateOnly)
}

type Stats struct {
	account        accounts.AccountID
	tilesTaken     uint64
	streak         Streak
	streakBest     uint32
	daysPlayed     uint64
	messagesSent   uint64
	bombsDropped   uint64
	tilesFortified uint64
	bonusesClaimed uint64
	rightAnswers   uint64
	missionsDone   uint64
	missionsByKind map[MissionKind]MissionTally
	forgivenDays   []Day
	counted        Counted
}

// MissionKind names a kind of mission as missions names it in its briefs: missions.v1.MissionKind without its prefix,
// in lower case.
type MissionKind string

// MissionTally is the missions of one kind an account has done in all, and their goals summed.
type MissionTally struct {
	Done  uint64
	Goals uint64
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
	Account        accounts.AccountID
	TilesTaken     uint64
	Streak         Streak
	StreakBest     uint32
	DaysPlayed     uint64
	MessagesSent   uint64
	BombsDropped   uint64
	TilesFortified uint64
	BonusesClaimed uint64
	RightAnswers   uint64
	MissionsDone   uint64
	MissionsByKind map[MissionKind]MissionTally
	ForgivenDays   []Day
	Counted        Counted
}

func (s StatsSnapshot) Restore() Stats {
	return Stats{
		account:        s.Account,
		tilesTaken:     s.TilesTaken,
		streak:         s.Streak,
		streakBest:     s.StreakBest,
		daysPlayed:     s.DaysPlayed,
		messagesSent:   s.MessagesSent,
		bombsDropped:   s.BombsDropped,
		tilesFortified: s.TilesFortified,
		bonusesClaimed: s.BonusesClaimed,
		rightAnswers:   s.RightAnswers,
		missionsDone:   s.MissionsDone,
		missionsByKind: talliesOf(s.MissionsByKind),
		forgivenDays:   daysOf(s.ForgivenDays),
		counted:        s.Counted,
	}
}

func (s Stats) TakeSnapshot() StatsSnapshot {
	return StatsSnapshot{
		Account:        s.account,
		TilesTaken:     s.tilesTaken,
		Streak:         s.streak,
		StreakBest:     s.streakBest,
		DaysPlayed:     s.daysPlayed,
		MessagesSent:   s.messagesSent,
		BombsDropped:   s.bombsDropped,
		TilesFortified: s.tilesFortified,
		BonusesClaimed: s.bonusesClaimed,
		RightAnswers:   s.rightAnswers,
		MissionsDone:   s.missionsDone,
		MissionsByKind: talliesOf(s.missionsByKind),
		ForgivenDays:   daysOf(s.forgivenDays),
		Counted:        s.counted,
	}
}

func NewStats(account accounts.AccountID) Stats {
	return Stats{account: account}
}

func AccountsOf(page []Stats) []accounts.AccountID {
	accounts := make([]accounts.AccountID, 0, len(page))
	for _, stats := range page {
		accounts = append(accounts, stats.Account())
	}
	return accounts
}

func (s Stats) Account() accounts.AccountID { return s.account }

func (s Stats) TilesTaken() uint64 { return s.tilesTaken }

func (s Stats) Streak() Streak { return s.streak }

func (s Stats) StreakBest() uint32 { return s.streakBest }

// DaysPlayed is the UTC days with at least one take: the streak's days, counted whether or not they follow each other.
func (s Stats) DaysPlayed() uint64 { return s.daysPlayed }

func (s Stats) MessagesSent() uint64 { return s.messagesSent }

func (s Stats) BombsDropped() uint64 { return s.bombsDropped }

func (s Stats) TilesFortified() uint64 { return s.tilesFortified }

func (s Stats) BonusesClaimed() uint64 { return s.bonusesClaimed }

func (s Stats) RightAnswers() uint64 { return s.rightAnswers }

// MissionsDone is how many missions the account was paid in all, which the Missions title track counts.
func (s Stats) MissionsDone() uint64 { return s.missionsDone }

// MissionGoals is the goals of the missions of the kind the account has done, summed, a guest's too.
func (s Stats) MissionGoals(kind MissionKind) uint64 { return s.missionsByKind[kind].Goals }

func (s Stats) MissionsByKind() map[MissionKind]MissionTally { return talliesOf(s.missionsByKind) }

// A copy, so no two stats share one map; none when it holds no kind, so a row read back equals the one recorded.
func talliesOf(tallies map[MissionKind]MissionTally) map[MissionKind]MissionTally {
	if len(tallies) == 0 {
		return nil
	}
	return maps.Clone(tallies)
}

// ForgivenDays is the days the operator forgave the account, in order: none of them is a day played.
func (s Stats) ForgivenDays() []Day { return daysOf(s.forgivenDays) }

func daysOf(days []Day) []Day {
	if len(days) == 0 {
		return nil
	}
	sorted := slices.Clone(days)
	slices.SortFunc(sorted, Day.Compare)
	return slices.Compact(sorted)
}

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

// Changes is each change in turn, as one: an account's takes of a batch of facts, recorded in one write.
func Changes(changes ...func(Stats) Stats) func(Stats) Stats {
	return func(s Stats) Stats {
		for _, change := range changes {
			s = change(s)
		}
		return s
	}
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

func MissionsDone(byKind map[MissionKind]MissionTally) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithMissionsDone(byKind) }
}

func PlayDays(days []Day) func(Stats) Stats {
	return func(s Stats) Stats { return s.WithPlayDays(days) }
}

// ForgivenDay leaves the stats as they are when the rule refuses the day under the lock: the caller asked it before.
func ForgivenDay(day Day, played []Day, today Day) func(Stats) Stats {
	return func(s Stats) Stats {
		forgiven, err := s.WithForgivenDay(day, played, today)
		if err != nil {
			return s
		}
		return forgiven
	}
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
	s.daysPlayed++

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

// Missions tells what was done in all, by kind, so a total told twice, late or lost changes nothing: each kind keeps the
// highest it heard.
func (s Stats) WithMissionsDone(byKind map[MissionKind]MissionTally) Stats {
	tallies := talliesOf(s.missionsByKind)
	for kind, told := range byKind {
		if told == (MissionTally{}) {
			continue
		}
		if tallies == nil {
			tallies = map[MissionKind]MissionTally{}
		}
		kept := tallies[kind]
		tallies[kind] = MissionTally{Done: max(kept.Done, told.Done), Goals: max(kept.Goals, told.Goals)}
	}
	s.missionsByKind = tallies
	return s
}

// WithPlayDays counts the days played again from the days the ledger holds a take on, for the backfill. Only the days up
// to the streak's last day count, and that day always does: a take the live count already counted on a later day is
// not counted twice, and one flushed after the ledger was read is not lost. The best streak less the days forgiven is a
// floor, for the takes from before the ledger had accounts; a forgiven day is never a day played, and the days played
// never go down.
func (s Stats) WithPlayDays(days []Day) Stats {
	last := s.streak.lastDay
	if last.Empty() {
		return s
	}
	forgiven := cpcolls.NewSet(s.forgivenDays...)
	played := cpcolls.NewSet[Day]()
	for _, day := range append([]Day{last}, days...) {
		if !last.Before(day) && !forgiven.Contains(day) {
			played.Add(day)
		}
	}
	best, daysForgiven := uint64(s.streakBest), uint64(len(s.forgivenDays))
	s.daysPlayed = max(uint64(played.Len()), best-min(daysForgiven, best), s.daysPlayed)
	return s
}

var (
	ErrNoStreak     = errors.New("the account never took a tile, so it has no streak")
	ErrNotPast      = errors.New("the day is today or later: the player can still play it")
	ErrDayForgiven  = errors.New("the day is forgiven already")
	ErrDayPlayed    = errors.New("the account played that day")
	ErrJoinsNothing = errors.New("the day joins nothing to the streak: it is more than one day after the last day played, " +
		"or apart from the streak's run")
)

// WithForgivenDay counts the streak as though the account had played the day, which is no take: no day played, no tile.
func (s Stats) WithForgivenDay(day Day, played []Day, today Day) (Stats, error) {
	last := s.streak.lastDay
	switch {
	case last.Empty():
		return s, ErrNoStreak
	case !day.Before(today):
		return s, ErrNotPast
	case slices.Contains(s.forgivenDays, day):
		return s, ErrDayForgiven
	case slices.Contains(played, day) || s.streak.Covers(day):
		return s, ErrDayPlayed
	case last.Following().Before(day):
		return s, ErrJoinsNothing
	}

	onTheStreak := cpcolls.NewSet(append(slices.Clone(played), s.forgivenDays...)...)
	for covered, n := last, uint32(0); n < max(s.streak.days, 1); covered, n = covered.Preceding(), n+1 {
		onTheStreak.Add(covered)
	}
	before := runOf(last, onTheStreak)
	onTheStreak.Add(day)
	end := last
	if last.Before(day) {
		end = day
	}
	run := runOf(end, onTheStreak)
	if end == last && run == before {
		return s, ErrJoinsNothing
	}

	s.streak = Streak{days: max(s.streak.days, run), lastDay: end}
	s.streakBest = max(s.streakBest, run)
	s.forgivenDays = daysOf(append(slices.Clone(s.forgivenDays), day))
	return s, nil
}

func runOf(end Day, days *cpcolls.Set[Day]) uint32 {
	var run uint32
	for day := end; days.Contains(day); day = day.Preceding() {
		run++
	}
	return run
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

func (s Streak) Covers(day Day) bool {
	if s.days == 0 || s.lastDay.Before(day) {
		return false
	}
	first := Day{midnight: s.lastDay.midnight.AddDate(0, 0, -int(s.days-1))}
	return !day.Before(first)
}
