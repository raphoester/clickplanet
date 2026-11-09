package hear_arena_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type Closed interface {
	KeepClosed(ctx context.Context, arena moments.RoundID) (bool, error)
}

type UseCase interface {
	// Execute answers whether the arena was not kept closed yet.
	Execute(ctx context.Context, arena moments.RoundID) (bool, error)
}

func New(closed Closed) *Hearing {
	return &Hearing{closed: closed}
}

// Hearing keeps each arena planet told closed: what waited for its map may open.
type Hearing struct {
	closed Closed
}

var _ UseCase = (*Hearing)(nil)

func (h *Hearing) Execute(ctx context.Context, arena moments.RoundID) (bool, error) {
	return h.closed.KeepClosed(ctx, arena) //nolint:wrapcheck // the store names it.
}
