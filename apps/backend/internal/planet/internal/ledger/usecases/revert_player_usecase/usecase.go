// Package revert_player_usecase gives back every tile one caller took and nobody has taken since.
package revert_player_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/pacing"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

type Ledger interface {
	TakenBy(scope string) []ledger.Taking
	Forget(takings []ledger.Taking)
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
	// Touched is every tile the ledger says this scope took last; Held is those still wearing its paint.
	Touched  int
	Held     int
	Restored int
}

func New(ledger Ledger, tiles Map, pace pacing.Pacing) *UseCase {
	return &UseCase{ledger: ledger, tiles: tiles, pacing: pace}
}

type UseCase struct {
	ledger Ledger
	tiles  Map
	pacing pacing.Pacing
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	scope, ok := cpipscope.Parse(in.Scope)
	if !ok {
		return Out{}, fmt.Errorf("%w: %q", clicks.ErrInvalidScope, in.Scope)
	}
	if u.pacing.Batch <= 0 {
		return Out{}, errors.New("revert batch must be positive")
	}

	takings := u.ledger.TakenBy(scope)
	out := Out{Scope: scope, Touched: len(takings)}

	restorations := make([]clicks.Restoration, 0, len(takings))
	for _, taking := range takings {
		if owner, _ := u.tiles.Owner(taking.Tile); owner == taking.Country {
			restorations = append(restorations, clicks.Restoration{Tile: taking.Tile, From: taking.Country, To: taking.Previous})
		}
	}
	out.Held = len(restorations)

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

	// Every take, covered ones included: none of them is this scope's to undo any more.
	u.ledger.Forget(takings)

	return out, nil
}
