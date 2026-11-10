package hear_opened_arena_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
)

type Opened interface {
	KeepOpened(ctx context.Context, arena moments.RoundID) (bool, error)
}

type UseCase interface {
	// Execute answers whether the arena was not kept opened yet.
	Execute(ctx context.Context, arena moments.RoundID) (bool, error)
}

func New(opened Opened) *Hearing {
	return &Hearing{opened: opened}
}

// Hearing keeps each arena planet told opened: its arena battle's round may open on the blank map.
type Hearing struct {
	opened Opened
}

var _ UseCase = (*Hearing)(nil)

func (h *Hearing) Execute(ctx context.Context, arena moments.RoundID) (bool, error) {
	return h.opened.KeepOpened(ctx, arena) //nolint:wrapcheck // the store names it.
}
