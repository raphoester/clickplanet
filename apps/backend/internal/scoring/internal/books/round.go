package books

import (
	"errors"
	"fmt"
	"time"
)

type Country string

// RoundID is the id another module gave a round: scoring only repeats it.
type RoundID string

func ParseRoundID(id string) (RoundID, error) {
	if err := checkID(id); err != nil {
		return "", fmt.Errorf("round %q: %w", id, err)
	}
	return RoundID(id), nil
}

const MaxTimes = 10

var ErrInvalidTimes = fmt.Errorf("a round scores 1 to %d times its points", MaxTimes)

// ErrRefused is a command that can never apply: delivered again, it would hold back the ones after it.
var ErrRefused = errors.New("refused")

// Round credits its book once scored: the countries ranked by the tiles they gained during it, times Times.
type Round struct {
	ID       RoundID
	Book     BookID
	Times    uint32
	StartsAt time.Time
	Scored   bool
	// EndsAt is a scored round's.
	EndsAt time.Time
}

func NewRound(id RoundID, book BookID, times uint32, startsAt time.Time) (Round, error) {
	if err := checkTimes(times); err != nil {
		return Round{}, fmt.Errorf("%w: round %q: %w", ErrRefused, id, err)
	}
	return Round{ID: id, Book: book, Times: times, StartsAt: startsAt.UTC()}, nil
}

func checkTimes(times uint32) error {
	if times < 1 || times > MaxTimes {
		return fmt.Errorf("%w, not %d", ErrInvalidTimes, times)
	}
	return nil
}

// Holds says whether a take at a time counts in the round's numbers: from its start, and before its end once scored.
func (r Round) Holds(at time.Time) bool {
	return !at.Before(r.StartsAt) && (!r.Scored || at.Before(r.EndsAt))
}

// ScoredAt is the round scored at its end from the map then, against the map it opened on.
func (r Round) ScoredAt(endsAt time.Time, start Start, end Snapshot) Scored {
	r.Scored, r.EndsAt = true, endsAt.UTC()
	return Scored{Round: r, Results: r.Results(start, end)}
}

type Snapshot struct {
	Tiles uint32
	Held  map[Country]uint32
}

// Start is the tiles each country held when a round opened; an empty one counts from nothing.
type Start map[Country]uint32
