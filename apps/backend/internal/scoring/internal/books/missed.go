package books

import (
	"errors"
	"time"
)

var ErrMissed = errors.New("a close missed it before it ever opened")

// MissedRound is a round closed before it ever opened: it never opens, and every close of it answers the first's end.
type MissedRound struct {
	ID     RoundID
	EndsAt time.Time
}

func NewMissedRound(id RoundID, endsAt time.Time) MissedRound {
	return MissedRound{ID: id, EndsAt: endsAt.UTC()}
}

func (m MissedRound) Ending() Ending {
	return Ending{At: m.EndsAt, Missed: true}
}

type MissedBook struct {
	ID     BookID
	EndsAt time.Time
}

func NewMissedBook(id BookID, endsAt time.Time) MissedBook {
	return MissedBook{ID: id, EndsAt: endsAt.UTC()}
}

func (m MissedBook) Ending() Ending {
	return Ending{At: m.EndsAt, Missed: true}
}

// Ending is what a close answers: the end a round or a book closed at, or the end of the close that missed it.
type Ending struct {
	At     time.Time
	Missed bool
}
