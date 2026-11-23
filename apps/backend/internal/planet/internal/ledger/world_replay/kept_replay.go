package world_replay

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

// Kept is the ledger memory keeps: its last 72 hours, from its head on.
type Kept interface {
	ReplayThrough(
		erased func(head ledger.Position) (*ledger.Erasure, error), see func(on clicks.Stage, event ledger.Event),
	) (ledger.Position, error)
}

type Erased interface {
	Before(ctx context.Context, position ledger.Position) (*ledger.Erasure, error)
}

func NewKeptReplay(kept Kept, erased Erased) KeptReplay {
	return KeptReplay{kept: kept, erased: erased}
}

// KeptReplay replays the ledger memory keeps through what a revert forgot before its head too.
type KeptReplay struct {
	kept   Kept
	erased Erased
}

func (k KeptReplay) Replay(ctx context.Context, see func(on clicks.Stage, event ledger.Event)) (ledger.Position, error) {
	return k.kept.ReplayThrough(func(head ledger.Position) (*ledger.Erasure, error) { //nolint:wrapcheck // Before names it.
		return k.erased.Before(ctx, head)
	}, see)
}
