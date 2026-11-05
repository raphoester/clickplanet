package hear_book_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Planner interface {
	Plan(ctx context.Context) (moments.Plan, error)
}

type Books interface {
	KeepBook(ctx context.Context, closed moments.ClosedBook) (bool, error)
}

// Heard is what came of a book scoring told.
type Heard uint8

const (
	// Told: the book was new, and the season's end is recorded.
	Told Heard = iota + 1
	// Repeated: the book was kept already.
	Repeated
	// Unknown: no season has this book.
	Unknown
)

type UseCase interface {
	Execute(ctx context.Context, closed moments.ClosedBook) (Heard, error)
}

func New(planner Planner, books Books, events moments.Events, transactioner cpatomic.Transactioner) *Hearing {
	return &Hearing{planner: planner, books: books, events: events, transactioner: transactioner}
}

// Hearing keeps each book scoring closed, and tells the season's end once.
type Hearing struct {
	planner       Planner
	books         Books
	events        moments.Events
	transactioner cpatomic.Transactioner
}

func (h *Hearing) Execute(ctx context.Context, closed moments.ClosedBook) (Heard, error) {
	plan, err := h.planner.Plan(ctx)
	if err != nil {
		return 0, err //nolint:wrapcheck // the planner names what it read.
	}
	moment, err := plan.BookClosed(closed.Book, closed.EndedAt)
	if errors.Is(err, moments.ErrUnknownBook) {
		return Unknown, nil
	}
	if err != nil {
		return 0, err //nolint:wrapcheck // the plan names the book.
	}

	heard := Repeated
	err = h.transactioner.InTx(ctx, func(ctx context.Context) error {
		kept, err := h.books.KeepBook(ctx, closed)
		if err != nil || !kept {
			return err //nolint:wrapcheck // the store names the book.
		}
		heard = Told
		return h.events.Record(ctx, moment) //nolint:wrapcheck // the adapter names the event.
	})
	if err != nil {
		return 0, err //nolint:wrapcheck // the transaction's own error.
	}
	return heard, nil
}
