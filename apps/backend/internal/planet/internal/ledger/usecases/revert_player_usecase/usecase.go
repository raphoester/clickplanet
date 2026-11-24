package revert_player_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Ledger interface {
	Forget(caller ledger.Caller, before ledger.Position)
}

// World reads the world's ledger from its last blank map, every board, and answers the position after the last event.
type World interface {
	Replay(ctx context.Context, start func(first clicks.Stage) ledger.Replayed) (ledger.Position, error)
}

// ErrArenaPlays is a revert asked while an arena's map plays: the world's waits aside, and Forget would hide its takes
// the revert did not undo.
var ErrArenaPlays = errors.New("a revert waits for the world's map: an arena's plays")

type Map interface {
	Stage() clicks.Stage
	Owner(tile uint32) (string, bool)
	Shields(tile uint32) int
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
	Shields  int
}

func New(ledger Ledger, world World, tiles Map, pace clicks.Pacing) *UseCase {
	return &UseCase{ledger: ledger, world: world, tiles: tiles, pacing: pace}
}

type UseCase struct {
	ledger Ledger
	world  World
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

	playing := u.tiles.Stage()
	if playing.Map != clicks.World {
		return Out{}, fmt.Errorf("%w: %s", ErrArenaPlays, playing.Map)
	}

	var undoing *ledger.Undoing
	end, err := u.world.Replay(ctx, func(first clicks.Stage) ledger.Replayed {
		undoing = ledger.NewUndoing(caller, first)
		return undoing
	})
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the world's ledger: %w", err)
	}

	restorations := undoing.Restorations(u.tiles, playing)
	out := Out{Scope: caller.Scope, Account: caller.Account, Touched: undoing.Touched(), Held: len(restorations)}
	for _, restoration := range restorations {
		out.Shields += restoration.ShieldsBack()
	}

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
