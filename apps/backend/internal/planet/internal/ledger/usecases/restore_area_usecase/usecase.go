package restore_area_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const shownPainters = 50

var (
	ErrNoTime        = errors.New("a restore needs the time to put the area back to")
	ErrPastTheLedger = errors.New("the ledger does not reach back that far")
)

type Ledger interface {
	Replay(see func(ledger.Event)) ledger.Position
}

type Map interface {
	Owner(tile uint32) (string, bool)
	Restore(ctx context.Context, restorations []clicks.Restoration) (int, error)
}

type Areas interface {
	Around(tile uint32, rings uint32) ([]uint32, error)
}

type In struct {
	Tile   uint32
	Rings  uint32
	Before time.Time
	DryRun bool
}

type Out struct {
	Tiles         int
	Changed       int
	Restored      int
	Painters      []ledger.Player
	PaintersTotal int
	KeptSince     time.Time
}

func New(ledger Ledger, tiles Map, areas Areas, reach ledger.Reach, pace clicks.Pacing) *UseCase {
	return &UseCase{ledger: ledger, tiles: tiles, areas: areas, reach: reach, pacing: pace}
}

type UseCase struct {
	ledger Ledger
	tiles  Map
	areas  Areas
	reach  ledger.Reach
	pacing clicks.Pacing
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if in.Before.IsZero() {
		return Out{}, ErrNoTime
	}
	if u.pacing.Batch <= 0 {
		return Out{}, errors.New("restore batch must be positive")
	}

	area, err := u.areas.Around(in.Tile, in.Rings)
	if err != nil {
		return Out{}, fmt.Errorf("cannot restore: %w", err)
	}

	keptSince := u.reach.Since()
	if in.Before.Before(keptSince) {
		return Out{}, fmt.Errorf("%w: it keeps from %s", ErrPastTheLedger, keptSince.Format(time.RFC3339))
	}

	// The ledger in memory keeps whole seconds: a take in the second asked for is after it.
	since := in.Before.Truncate(time.Second)
	inside := cpcolls.NewSet(area...)
	rewind := ledger.NewRewind(area, since)
	painters := ledger.NewTally(func(taking ledger.Taking) bool {
		return !taking.At.Before(since) && inside.Contains(taking.Tile)
	})
	u.ledger.Replay(func(event ledger.Event) {
		rewind.See(event)
		painters.See(event)
	})

	restorations := rewind.Restorations(u.tiles)
	everyone := ledger.ByTakes(painters.Players(u.tiles))
	out := Out{
		Tiles: len(area), Changed: len(restorations), KeptSince: keptSince,
		Painters: ledger.Top(everyone, shownPainters), PaintersTotal: len(everyone),
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
				return out, fmt.Errorf("restore %w", err)
			}
		}
	}

	return out, nil
}
