package replay_usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrInvalidWindow = errors.New("a replay needs a since before its until")
	ErrBeforeReset   = errors.New("the map was reset after this window ended: the map it played on is gone")
)

type Ledger interface {
	Replay(see func(ledger.Event)) ledger.Position
}

type Resets interface {
	After(ctx context.Context, at time.Time) ([]resets.Reset, error)
}

type Arenas interface {
	Stretches(ctx context.Context) ([]arenas.Stretch, error)
}

// Tiles clamps each range to the map's last tile.
type Tiles interface {
	StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error)
	Kept(ctx context.Context, kept clicks.MapID) (clicks.DenseBatch, error)
}

type In struct {
	Since time.Time
	Until time.Time
}

type Out struct {
	Since   time.Time
	Until   time.Time
	Opening clicks.DenseBatch
	Scenes  []ledger.Scene
	Asides  []arenas.Stretch
}

func New(ledger Ledger, resets Resets, arenas Arenas, tiles Tiles, clock cptime.Clock) *UseCase {
	return &UseCase{ledger: ledger, resets: resets, arenas: arenas, tiles: tiles, clock: clock}
}

type UseCase struct {
	ledger Ledger
	resets Resets
	arenas Arenas
	tiles  Tiles
	clock  cptime.Clock
}

// Execute rewinds the world's map from now: a reset in the window starts the replay at it, on the blank map, and a
// window that ended before a reset cannot be rewound to. An arena's scenes are left out: the world's map came back.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	until := in.Until
	if until.IsZero() {
		until = u.clock.Now()
	}
	if in.Since.IsZero() || !in.Since.Before(until) {
		return Out{}, ErrInvalidWindow
	}
	after, err := u.resets.After(ctx, in.Since)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the map's resets: %w", err)
	}
	for _, reset := range after {
		if reset.At.After(until) {
			return Out{}, fmt.Errorf("%w: %q at %s", ErrBeforeReset, reset.ID, reset.At.Format(time.RFC3339))
		}
		in.Since = reset.At
	}
	stretches, err := u.arenas.Stretches(ctx)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the arenas: %w", err)
	}
	maps := arenas.MapsOf(stretches)

	// The map before the ledger: a change made between the two reads is then in the ledger, and rewound.
	now, err := u.world(ctx, maps)
	if err != nil {
		return Out{}, err
	}

	footage := ledger.NewFootage(in.Since, until, maps.World())
	u.ledger.Replay(footage.See)

	return Out{
		Since: in.Since, Until: until, Opening: footage.Opening(now), Scenes: footage.Scenes(),
		Asides: maps.Asides(in.Since, until),
	}, nil
}

func (u *UseCase) world(ctx context.Context, maps arenas.Maps) (clicks.DenseBatch, error) {
	if _, open := maps.Open(); open {
		kept, err := u.tiles.Kept(ctx, clicks.World)
		if err != nil {
			return clicks.DenseBatch{}, fmt.Errorf("failed to read the world's map kept: %w", err)
		}
		return kept, nil
	}
	now, err := u.tiles.StateBatchDense(1, math.MaxUint32)
	if err != nil {
		return clicks.DenseBatch{}, fmt.Errorf("failed to read the map: %w", err)
	}
	return now, nil
}
