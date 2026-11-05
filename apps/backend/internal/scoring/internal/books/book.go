package books

import (
	"errors"
	"fmt"
	"time"
)

const maxID = 64

var ErrInvalidID = errors.New("an id is 1 to 64 characters")

// BookID is the id another module gave a book: scoring only repeats it.
type BookID string

func ParseBookID(id string) (BookID, error) {
	if err := checkID(id); err != nil {
		return "", fmt.Errorf("book %q: %w", id, err)
	}
	return BookID(id), nil
}

func checkID(id string) error {
	if id == "" || len(id) > maxID {
		return ErrInvalidID
	}
	return nil
}

// Book gathers the points of its rounds and grants and each player's takes, from its start until it is closed.
type Book struct {
	ID       BookID
	StartsAt time.Time
	Closed   bool
	// EndsAt and Tiebreak are a closed book's; the zero Tiebreak is none.
	EndsAt   time.Time
	Tiebreak RoundID
}

func NewBook(id BookID, startsAt time.Time) Book {
	return Book{ID: id, StartsAt: startsAt.UTC()}
}

// Holds says whether a take at a time counts in the book: from its start, and before its end once closed.
func (b Book) Holds(at time.Time) bool {
	return !at.Before(b.StartsAt) && (!b.Closed || at.Before(b.EndsAt))
}

// ClosedAt is the book closed at its end, ranked by its rounds with tiebreak as the third key.
func (b Book) ClosedAt(endsAt time.Time, tiebreak RoundID) Book {
	b.Closed, b.EndsAt, b.Tiebreak = true, endsAt.UTC(), tiebreak
	return b
}

// Table ranks the book from the rounds it scored and the grants it holds.
func (b Book) Table(scored []Scored, grants []Grant) []Placed {
	var held []Grant
	for _, grant := range grants {
		if b.Holds(grant.At) {
			held = append(held, grant)
		}
	}
	return TableOf(ScoresOf(scored, held, b.Tiebreak))
}
