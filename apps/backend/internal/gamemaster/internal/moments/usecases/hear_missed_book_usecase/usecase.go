package hear_missed_book_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type Missed interface {
	KeepMissedBook(ctx context.Context, missed moments.MissedBook) (bool, error)
}

type UseCase interface {
	// Execute answers whether the book was not kept missed yet.
	Execute(ctx context.Context, missed moments.MissedBook) (bool, error)
}

func New(missed Missed) *Hearing {
	return &Hearing{missed: missed}
}

// Hearing keeps each book scoring told missed, so it is closed no more; the players are told nothing.
type Hearing struct {
	missed Missed
}

var _ UseCase = (*Hearing)(nil)

func (h *Hearing) Execute(ctx context.Context, missed moments.MissedBook) (bool, error) {
	return h.missed.KeepMissedBook(ctx, missed) //nolint:wrapcheck // the store names it.
}
