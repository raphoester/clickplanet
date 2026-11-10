package moments

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

var (
	ErrOffCutoff = errors.New("a battle starts or ends at the daily cutoff")
	ErrOverlap   = errors.New("a battle may not overlap another battle or a season's finale")
	ErrDayTaken  = errors.New("a battle may not take a whole day")
)

// Battle is a round of a season's book with a battle's look: an entry of kind battle that starts while the season
// runs, cut at its end, or the season's finale.
type Battle struct {
	// Entry is the zero id for the finale, which is the season's and no entry's.
	Entry  schedule.EntryID
	Season season.Number
	// Number is the battle's place among the season's battles, by start.
	Number      uint32
	Times       schedule.Times
	Description schedule.Description
	StartsAt    time.Time
	EndsAt      time.Time
	// EndsSeason is true for the season's finale alone.
	EndsSeason bool
	// Set is the rule set it plays: the entry's, or the finale's.
	Set schedule.SetName
}

func (b Battle) Round() RoundID {
	if b.EndsSeason {
		return FinaleRoundOf(b.Season)
	}
	return BattleRoundOf(b.Entry)
}

func (b Battle) runs(at time.Time) bool {
	return !at.Before(b.StartsAt) && at.Before(b.EndsAt)
}

func (b Battle) String() string {
	window := fmt.Sprintf("from %s to %s", b.StartsAt.UTC().Format(time.RFC3339), b.EndsAt.UTC().Format(time.RFC3339))
	if b.EndsSeason {
		return fmt.Sprintf("season %d's finale, %s", b.Season, window)
	}
	return fmt.Sprintf("the battle %s, %s", b.Entry, window)
}

// Book is a season's book: its days and its battles.
type Book struct {
	Window  season.Window
	Days    []season.Day
	Battles []Battle
}

func (b Book) ID() BookID { return BookOf(b.Window.Number) }

// Tiebreak is the round that breaks a tie: the season's finale, or none.
func (b Book) Tiebreak() RoundID {
	if battle, ok := b.FinaleBattle(); ok {
		return battle.Round()
	}
	return ""
}

// FinaleBattle is the season's finale, when it has one.
func (b Book) FinaleBattle() (Battle, bool) {
	for _, battle := range b.Battles {
		if battle.EndsSeason {
			return battle, true
		}
	}
	return Battle{}, false
}

type round struct {
	id       RoundID
	times    schedule.Times
	startsAt time.Time
	endsAt   time.Time
	battle   bool
}

func (b Book) rounds() []round {
	rounds := make([]round, 0, len(b.Days)+len(b.Battles))
	for _, day := range b.Days {
		rounds = append(rounds, round{id: DayRoundOf(day), times: 1, startsAt: day.StartsAt, endsAt: day.EndsAt})
	}
	for _, battle := range b.Battles {
		rounds = append(rounds, round{
			id: battle.Round(), times: battle.Times, startsAt: battle.StartsAt, endsAt: battle.EndsAt, battle: true,
		})
	}
	return rounds
}

// Plan is every book scoring keeps for the gamemaster: one per season, its days at the daily cutoff and its battles.
type Plan struct {
	books  []Book
	cutoff season.Cutoff
	known  Known
}

// A day that opened keeps running to the season's end when the end moves before its cutoff: opened names the rounds
// the gamemaster opened.
func NewPlan(calendar season.Calendar, cutoff season.Cutoff, opened []RoundID, entries ...schedule.Entry) Plan {
	battles := make([]schedule.EntrySnapshot, 0, len(entries))
	for _, entry := range entries {
		if snapshot := entry.TakeSnapshot(); snapshot.Kind == schedule.Battle && snapshot.Times >= 1 {
			battles = append(battles, snapshot)
		}
	}
	slices.SortFunc(battles, func(a, b schedule.EntrySnapshot) int {
		return cmp.Or(a.StartsAt.Compare(b.StartsAt), a.MadeAt.Compare(b.MadeAt), bytes.Compare(a.ID[:], b.ID[:]))
	})

	windows := calendar.Windows()
	books := make([]Book, 0, len(windows))
	for _, window := range windows {
		book := Book{Window: window, Days: window.Days(cutoff)}
		if cut, ok := window.CutDay(cutoff); ok && slices.Contains(opened, DayRoundOf(cut)) {
			book.Days = append(book.Days, cut)
		}
		for _, entry := range battles {
			if !window.Runs(entry.StartsAt) {
				continue
			}
			ends := entry.EndsAt.UTC()
			if ends.After(window.EndsAt) {
				ends = window.EndsAt
			}
			book.Battles = append(book.Battles, Battle{
				Entry: entry.ID, Season: window.Number, Times: entry.Times, Description: entry.Description,
				StartsAt: entry.StartsAt.UTC(), EndsAt: ends, Set: entry.Set,
			})
		}
		if window.Finale.Planned() {
			book.Battles = append(book.Battles, Battle{
				Season: window.Number, Times: window.Finale.Times, Description: window.Finale.Description,
				StartsAt: window.FinaleStartsAt(), EndsAt: window.EndsAt, EndsSeason: true, Set: window.Finale.Set,
			})
		}
		slices.SortStableFunc(book.Battles, func(a, b Battle) int { return a.StartsAt.Compare(b.StartsAt) })
		for i := range book.Battles {
			book.Battles[i].Number = uint32(i + 1) //nolint:gosec // a season holds far fewer battles.
		}
		for i, day := range book.Days {
			book.Days[i] = around(day, book.Battles)
		}
		books = append(books, book)
	}
	return Plan{books: books, cutoff: cutoff}
}

func around(day season.Day, battles []Battle) season.Day {
	starts, ends := day.StartsAt, day.EndsAt
	for _, battle := range battles {
		if battle.runs(starts) {
			day.StartsAt = battle.EndsAt
		}
		if battle.StartsAt.Before(ends) && !battle.EndsAt.Before(ends) {
			day.EndsAt = battle.StartsAt
		}
	}
	return day
}

func (p Plan) Clashes(now time.Time) error {
	var found []error
	for _, book := range p.books {
		for i, battle := range book.Battles {
			if !battle.EndsAt.After(now) {
				continue
			}
			if !p.cutoff.Is(battle.StartsAt) && !p.cutoff.Is(battle.EndsAt) {
				found = append(found, fmt.Errorf("%w, %s: %s", ErrOffCutoff, p.cutoff, battle))
			}
			for _, other := range book.Battles[i+1:] {
				if other.StartsAt.Before(battle.EndsAt) && battle.StartsAt.Before(other.EndsAt) {
					found = append(found, fmt.Errorf("%w: %s, and %s", ErrOverlap, battle, other))
				}
			}
		}
		for _, day := range book.Days {
			if day.EndsAt.After(now) && !day.StartsAt.Before(day.EndsAt) {
				found = append(found, fmt.Errorf("%w: day %d of season %d", ErrDayTaken, day.Number, day.Season))
			}
		}
	}
	return errors.Join(found...)
}

// DueAt is every command whose time has come: a book or a round opened while it runs, and closed once it ended until
// scoring answered it, for every season that started. By time, and at one instant OpenBook, CloseRound, OpenRound, then
// CloseBook. With what was commanded known, a battle's round opens once it is judged, an arena battle's counting from
// nothing, nothing opens while an arena battle that ended before it waits for its map, and a round or a book scoring
// answered, scored, closed or missed, is closed no more.
func (p Plan) DueAt(now time.Time) []Command {
	var due []Command
	for _, book := range p.books {
		if !book.Window.Started(now) {
			continue
		}
		if !book.Window.Ended(now) && !p.Waits(book.Window.StartsAt) {
			due = append(due, Command{Action: OpenBook, Book: book.ID(), At: book.Window.StartsAt})
		}
		for _, r := range book.rounds() {
			switch {
			case now.Before(r.startsAt):
			case now.Before(r.endsAt):
				if open, ok := p.opening(book, r); ok {
					due = append(due, open)
				}
			case !p.settled(book.ID(), r.id):
				due = append(due, Command{Action: CloseRound, Book: book.ID(), Round: r.id, At: r.endsAt})
			}
		}
		if book.Window.Ended(now) && !p.known.books.Contains(book.ID()) {
			due = append(due, Command{Action: CloseBook, Book: book.ID(), Tiebreak: book.Tiebreak(), At: book.Window.EndsAt})
		}
	}
	slices.SortStableFunc(due, func(a, b Command) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.Action, b.Action))
	})
	return due
}

func (p Plan) opening(book Book, r round) (Command, bool) {
	open := Command{Action: OpenRound, Book: book.ID(), Round: r.id, Times: r.times, At: r.startsAt}
	if p.known.modes == nil {
		return open, true
	}
	mode := p.known.modes[r.id]
	if (r.battle && mode == 0) || p.Waits(r.startsAt) {
		return Command{}, false
	}
	open.FromNothing = mode == Arena
	return open, true
}

// ResetAt is the season that runs at a time and starts on a blank map, once its map may be wiped: scoring answered the
// book of the season before it, so that season's last round was read on the map before the wipe, and no arena battle
// before it waits for its map, which would put the old season's map back.
func (p Plan) ResetAt(at time.Time) (season.Window, bool) {
	book, ok := p.BookAt(at)
	running := book.Window
	if !ok || !running.StartsOnBlankMap() || p.Waits(running.StartsAt) {
		return season.Window{}, false
	}
	for _, before := range p.books {
		if before.Window.Number+1 == running.Number && !p.known.books.Contains(before.ID()) {
			return season.Window{}, false
		}
	}
	return running, true
}

// BookAt is the book of the season that runs at a time.
func (p Plan) BookAt(at time.Time) (Book, bool) {
	for _, book := range p.books {
		if book.Window.Runs(at) {
			return book, true
		}
	}
	return Book{}, false
}

// DayAt is the day that runs at a time: none between two seasons, nor from a season's last cutoff to its end, nor
// while an arena battle before it waits for its map.
func (p Plan) DayAt(at time.Time) (season.Day, bool) {
	book, ok := p.BookAt(at)
	if !ok {
		return season.Day{}, false
	}
	for _, day := range book.Days {
		if day.Runs(at) && !p.Waits(day.StartsAt) {
			return day, true
		}
	}
	return season.Day{}, false
}

// BattleAt is the battle that runs at a time, the season's finale included.
func (p Plan) BattleAt(at time.Time) (Battle, bool) {
	book, ok := p.BookAt(at)
	if !ok {
		return Battle{}, false
	}
	for _, battle := range book.Battles {
		if battle.runs(at) {
			return battle, true
		}
	}
	return Battle{}, false
}

func (p Plan) DaysOf(number season.Number) []season.Day {
	for _, book := range p.books {
		if book.Window.Number == number {
			return book.Days
		}
	}
	return nil
}

// BattleOf is the battle of an entry: none for an entry that is no battle, or that starts while no season runs.
func (p Plan) BattleOf(entry schedule.EntryID) (Battle, bool) {
	for _, book := range p.books {
		for _, battle := range book.Battles {
			if !battle.EndsSeason && battle.Entry == entry {
				return battle, true
			}
		}
	}
	return Battle{}, false
}

// FinaleAt is the finale that runs at a time.
func (p Plan) FinaleAt(at time.Time) (Battle, bool) {
	book, ok := p.BookAt(at)
	if !ok {
		return Battle{}, false
	}
	finale, ok := book.FinaleBattle()
	if !ok || at.Before(finale.StartsAt) {
		return Battle{}, false
	}
	return finale, true
}

// FinalesAfter are the finales that have not started at a time, by start.
func (p Plan) FinalesAfter(at time.Time) []Battle {
	var finales []Battle
	for _, book := range p.books {
		if finale, ok := book.FinaleBattle(); ok && finale.StartsAt.After(at) {
			finales = append(finales, finale)
		}
	}
	return finales
}

// RoundEnded is what a round scoring scored was, in the game's words.
func (p Plan) RoundEnded(id RoundID, endedAt time.Time) (Moment, error) {
	named, err := NamedOf(id)
	if err != nil {
		return nil, err
	}
	if named.battle() {
		if battle, ok := p.BattleOf(named.Entry); ok {
			return BattleEnded{Battle: battle, EndedAt: endedAt}, nil
		}
		return nil, fmt.Errorf("%w: %q", ErrUnknownRound, id)
	}
	book, err := p.book(named.Season)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrUnknownRound, id)
	}
	if named.Finale {
		if finale, ok := book.FinaleBattle(); ok {
			return BattleEnded{Battle: finale, EndedAt: endedAt}, nil
		}
		return nil, fmt.Errorf("%w: %q", ErrUnknownRound, id)
	}
	for _, day := range book.Days {
		if day.Number == named.Day {
			return DayEnded{Day: day, EndedAt: endedAt}, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownRound, id)
}

// BookClosed is the end of the season a book scoring closed was.
func (p Plan) BookClosed(id BookID, endedAt time.Time) (Moment, error) {
	number, err := SeasonOf(id)
	if err != nil {
		return nil, err
	}
	book, err := p.book(number)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrUnknownBook, id)
	}
	ended := SeasonEnded{Season: number, Book: id, Finale: book.Tiebreak(), EndedAt: endedAt}
	if battle, ok := book.FinaleBattle(); ok {
		ended.FinaleBattle = &BattleEnded{Battle: battle, EndedAt: battle.EndsAt}
	}
	return ended, nil
}

func (p Plan) book(number season.Number) (Book, error) {
	for _, book := range p.books {
		if book.Window.Number == number {
			return book, nil
		}
	}
	return Book{}, fmt.Errorf("season %d: %w", number, season.ErrUnknownSeason)
}
