package hear_missed_round_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type Missed interface {
	KeepMissedRound(ctx context.Context, missed moments.MissedRound) (bool, error)
}

type UseCase interface {
	// Execute answers whether the round was not kept missed yet.
	Execute(ctx context.Context, missed moments.MissedRound) (bool, error)
}

func New(missed Missed) *Hearing {
	return &Hearing{missed: missed}
}

// Hearing keeps each round scoring told missed, so it is closed no more; the players are told nothing.
type Hearing struct {
	missed Missed
}

var _ UseCase = (*Hearing)(nil)

func (h *Hearing) Execute(ctx context.Context, missed moments.MissedRound) (bool, error) {
	return h.missed.KeepMissedRound(ctx, missed) //nolint:wrapcheck // the store names it.
}
