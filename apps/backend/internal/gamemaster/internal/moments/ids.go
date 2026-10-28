package moments

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

// BookID names a season's book in scoring: season-N.
type BookID string

// RoundID names a round in scoring: season-N/day-K for a day, season-N/finale for a season's finale, the entry's id for
// another battle.
type RoundID string

var (
	ErrUnknownBook  = errors.New("no season has this book")
	ErrUnknownRound = errors.New("no day or battle has this round")
)

var (
	bookPattern   = regexp.MustCompile(`^season-(0|[1-9][0-9]{0,9})$`)
	dayPattern    = regexp.MustCompile(`^season-(0|[1-9][0-9]{0,9})/day-([1-9][0-9]{0,9})$`)
	finalePattern = regexp.MustCompile(`^season-(0|[1-9][0-9]{0,9})/finale$`)
)

func BookOf(number season.Number) BookID {
	return BookID(fmt.Sprintf("season-%d", number))
}

func DayRoundOf(day season.Day) RoundID {
	return RoundID(fmt.Sprintf("season-%d/day-%d", day.Season, day.Number))
}

func FinaleRoundOf(number season.Number) RoundID {
	return RoundID(fmt.Sprintf("season-%d/finale", number))
}

func BattleRoundOf(entry schedule.EntryID) RoundID {
	return RoundID(entry.String())
}

// SeasonOf is the season a book of the gamemaster's names.
func SeasonOf(book BookID) (season.Number, error) {
	match := bookPattern.FindStringSubmatch(string(book))
	if match == nil {
		return 0, fmt.Errorf("%w: %q", ErrUnknownBook, book)
	}
	number, err := numberOf(match[1])
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrUnknownBook, book)
	}
	return season.Number(number), nil
}

// Named is what a round of the gamemaster's names: a season's day, a season's finale, or a battle's entry.
type Named struct {
	Season season.Number
	Day    season.DayNumber
	Finale bool
	Entry  schedule.EntryID
}

func (n Named) battle() bool { return n.Day == 0 && !n.Finale }

func NamedOf(round RoundID) (Named, error) {
	if match := finalePattern.FindStringSubmatch(string(round)); match != nil {
		number, err := numberOf(match[1])
		if err != nil {
			return Named{}, fmt.Errorf("%w: %q", ErrUnknownRound, round)
		}
		return Named{Season: season.Number(number), Finale: true}, nil
	}
	if match := dayPattern.FindStringSubmatch(string(round)); match != nil {
		number, err := numberOf(match[1])
		if err != nil {
			return Named{}, fmt.Errorf("%w: %q", ErrUnknownRound, round)
		}
		day, err := numberOf(match[2])
		if err != nil {
			return Named{}, fmt.Errorf("%w: %q", ErrUnknownRound, round)
		}
		return Named{Season: season.Number(number), Day: season.DayNumber(day)}, nil
	}
	entry, err := schedule.ParseEntryID(string(round))
	if err != nil || entry.String() != string(round) {
		return Named{}, fmt.Errorf("%w: %q", ErrUnknownRound, round)
	}
	return Named{Entry: entry}, nil
}

func numberOf(digits string) (uint32, error) {
	number, err := strconv.ParseUint(digits, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%q is no number: %w", digits, err)
	}
	return uint32(number), nil
}
