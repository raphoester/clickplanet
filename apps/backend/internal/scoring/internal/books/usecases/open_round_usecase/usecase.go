package open_round_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Books interface {
	Find(ctx context.Context, id books.BookID) (books.Book, bool, error)
}

type Rounds interface {
	Find(ctx context.Context, id books.RoundID) (books.Round, bool, error)
	Open(ctx context.Context, round books.Round) (bool, error)
}

var ErrBookNotOpen = errors.New("the round's book is not open")

type Command struct {
	Round    books.RoundID
	Book     books.BookID
	Times    uint32
	StartsAt time.Time
}

func New(books Books, rounds Rounds) *UseCase {
	return &UseCase{books: books, rounds: rounds}
}

type UseCase struct {
	books  Books
	rounds Rounds
}

// Execute answers whether it opened the round: one kept before is left as it is, and one whose book is not open is
// refused.
func (u *UseCase) Execute(ctx context.Context, command Command) (bool, error) {
	round, err := books.NewRound(command.Round, command.Book, command.Times, command.StartsAt)
	if err != nil {
		return false, err //nolint:wrapcheck // the round says why.
	}
	if _, found, err := u.rounds.Find(ctx, command.Round); err != nil || found {
		return false, err //nolint:wrapcheck // the store says what failed.
	}
	book, found, err := u.books.Find(ctx, command.Book)
	if err != nil {
		return false, err //nolint:wrapcheck // the store says what failed.
	}
	if !found || book.Closed {
		return false, fmt.Errorf("%w: round %s: %w: %s", books.ErrRefused, command.Round, ErrBookNotOpen, command.Book)
	}
	opened, err := u.rounds.Open(ctx, round)
	if err != nil {
		return false, fmt.Errorf("failed to open round %s: %w", command.Round, err)
	}
	return opened, nil
}
