package revert_player_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	Replay(see func(on clicks.Stage, event ledger.Event)) ledger.Position
	Forget(caller ledger.Caller, before ledger.Position)
}

// ErrArenaPlays is a revert asked while an arena's map plays: the world's waits aside, and Forget would hide its takes
// the revert did not undo.
var ErrArenaPlays = errors.New("a revert waits for the world's map: an arena's plays")

type Map interface {
	Stage() clicks.Stage
	Owner(tile uint32) (string, bool)
	Restore(ctx context.Context, restorations []clicks.Restoration) (int, error)
}

type In struct {
	Scope   string
	Account string
	DryRun  bool
}

type Out struct {
	Scope    string
	Account  string
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
	caller, err := ledger.ParseCaller(in.Scope, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot revert: %w", err)
	}
	if u.pacing.Batch <= 0 {
		return Out{}, errors.New("revert batch must be positive")
	}

	// Only the events of the stage that plays, the world's map on its board, as each names it.
	playing := u.tiles.Stage()
	if playing.Map != clicks.World {
		return Out{}, fmt.Errorf("%w: %s", ErrArenaPlays, playing.Map)
	}

	runs := ledger.NewRuns(caller, playing)
	end := u.ledger.Replay(runs.See)

	restorations := runs.Restorations(u.tiles)
	out := Out{Scope: caller.Scope, Account: caller.Account, Touched: runs.Touched(), Held: len(restorations)}

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

	u.ledger.Forget(caller, end)

	return out, nil
}
