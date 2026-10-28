package moments

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

// Battle is an entry of kind battle that starts while a season runs: a round of that season's book, cut at its end.
type Battle struct {
	Entry  schedule.EntryID
	Season season.Number
	// Number is the battle's place among the season's battles, by start.
	Number     uint32
	Times      schedule.Times
	StartsAt   time.Time
	EndsAt     time.Time
	EndsSeason bool
}

func (b Battle) Round() RoundID { return BattleRoundOf(b.Entry) }

// Book is a season's book: its days and its battles.
type Book struct {
	Window  season.Window
	Days    []season.Day
	Battles []Battle
}

func (b Book) ID() BookID { return BookOf(b.Window.Number) }

// Finale is the round that ends at the season's end, which breaks a tie: the battle if one does, else a day, else none.
func (b Book) Finale() RoundID {
	if battle, ok := b.FinaleBattle(); ok {
		return battle.Round()
	}
	if last := len(b.Days) - 1; last >= 0 && b.Days[last].EndsAt.Equal(b.Window.EndsAt) {
		return DayRoundOf(b.Days[last])
	}
	return ""
}

// FinaleBattle is the battle that ends at the season's end, when one does.
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
}

func (b Book) rounds() []round {
	rounds := make([]round, 0, len(b.Days)+len(b.Battles))
	for _, day := range b.Days {
		rounds = append(rounds, round{id: DayRoundOf(day), times: 1, startsAt: day.StartsAt, endsAt: day.EndsAt})
	}
	for _, battle := range b.Battles {
		rounds = append(rounds, round{id: battle.Round(), times: battle.Times, startsAt: battle.StartsAt, endsAt: battle.EndsAt})
	}
	return rounds
}

// Plan is every book scoring keeps for the gamemaster: one per season, its days at the daily cutoff and its battles.
type Plan struct {
	books []Book
}

// A day that opened keeps running to the season's end when the end moves before its cutoff: opened names the rounds
// the gamemaster opened.
func NewPlan(calendar season.Calendar, cutoff season.Cutoff, opened []RoundID, entries ...schedule.EntrySnapshot) Plan {
	battles := slices.DeleteFunc(slices.Clone(entries), func(e schedule.EntrySnapshot) bool {
		return e.Kind != schedule.Battle || e.Times < 1
	})
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
				Entry: entry.ID, Season: window.Number, Number: uint32(len(book.Battles) + 1), Times: entry.Times,
				StartsAt: entry.StartsAt.UTC(), EndsAt: ends, EndsSeason: ends.Equal(window.EndsAt),
			})
		}
		books = append(books, book)
	}
	return Plan{books: books}
}

// DueAt is every command whose time has come: a book or a round opened while it runs, and closed once it ended, for
// every season that started. By time, and at one instant OpenBook, CloseRound, OpenRound, then CloseBook.
func (p Plan) DueAt(now time.Time) []Command {
	var due []Command
	for _, book := range p.books {
		if !book.Window.Started(now) {
			continue
		}
		if !book.Window.Ended(now) {
			due = append(due, Command{Action: OpenBook, Book: book.ID(), At: book.Window.StartsAt})
		}
		for _, r := range book.rounds() {
			switch {
			case now.Before(r.startsAt):
			case now.Before(r.endsAt):
				due = append(due, Command{Action: OpenRound, Book: book.ID(), Round: r.id, Times: r.times, At: r.startsAt})
			default:
				due = append(due, Command{Action: CloseRound, Book: book.ID(), Round: r.id, At: r.endsAt})
			}
		}
		if book.Window.Ended(now) {
			due = append(due, Command{Action: CloseBook, Book: book.ID(), Tiebreak: book.Finale(), At: book.Window.EndsAt})
		}
	}
	slices.SortStableFunc(due, func(a, b Command) int {
		return cmp.Or(a.At.Compare(b.At), cmp.Compare(a.Action, b.Action))
	})
	return due
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

// DayAt is the day that runs at a time: none between two seasons, nor from a season's last cutoff to its end.
func (p Plan) DayAt(at time.Time) (season.Day, bool) {
	book, ok := p.BookAt(at)
	if !ok {
		return season.Day{}, false
	}
	for _, day := range book.Days {
		if day.Runs(at) {
			return day, true
		}
	}
	return season.Day{}, false
}

// BattleOf is the battle of an entry: none for an entry that is no battle, or that starts while no season runs.
func (p Plan) BattleOf(entry schedule.EntryID) (Battle, bool) {
	for _, book := range p.books {
		for _, battle := range book.Battles {
			if battle.Entry == entry {
				return battle, true
			}
		}
	}
	return Battle{}, false
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
	ended := SeasonEnded{Season: number, Book: id, Finale: book.Finale(), EndedAt: endedAt}
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
