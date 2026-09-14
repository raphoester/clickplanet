// Package revert_player_usecase gives back every tile one caller still holds, to what it held before the caller's run on it.
package revert_player_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

type Ledger interface {
	Replay(see func(ledger.Taking)) ledger.Position
	Forget(scope string, before ledger.Position)
}

type Map interface {
	Owner(tile uint32) (string, bool)
	Restore(ctx context.Context, restorations []clicks.Restoration) (int, error)
}

type In struct {
	Scope  string
	DryRun bool
}

type Out struct {
	Scope string
	// Touched is every tile this scope took inside the retention; Held is those it still holds.
	Touched  int
	Held     int
	Restored int
}

func New(ledger Ledger, tiles Map, pace clicks.Pacing) *UseCase {
	return &UseCase{ledger: ledger, tiles: tiles, pacing: pace}
}

type UseCase struct {
	ledger Ledger
	tiles  Map
	pacing clicks.Pacing
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	scope, ok := cpipscope.Parse(in.Scope)
	if !ok {
		return Out{}, fmt.Errorf("%w: %q", ledger.ErrInvalidScope, in.Scope)
	}
	if u.pacing.Batch <= 0 {
		return Out{}, errors.New("revert batch must be positive")
	}

	runs := ledger.NewRuns(scope)
	end := u.ledger.Replay(runs.See)

	restorations := runs.Restorations(u.tiles)
	out := Out{Scope: scope, Touched: runs.Touched(), Held: len(restorations)}

	if in.DryRun {
		return out, nil
	}

	for len(restorations) > 0 {
		batch := restorations[:min(u.pacing.Batch, len(restorations))]
		restorations = restorations[len(batch):]

		restored, err := u.tiles.Restore(ctx, batch)
		out.Restored += restored
		if err != nil {
			return out, fmt.Errorf("failed to restore tiles: %w", err)
		}

		if len(restorations) > 0 {
			if err := u.pacing.Wait(ctx); err != nil {
				return out, fmt.Errorf("revert %w", err)
			}
		}
	}

	// Every take up to the replay, covered ones included: none of them is this scope's to undo any more.
	u.ledger.Forget(scope, end)

	return out, nil
}
