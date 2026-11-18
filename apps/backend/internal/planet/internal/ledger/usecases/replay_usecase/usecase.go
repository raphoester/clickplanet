package replay_usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	ErrInvalidWindow = errors.New("a replay needs a since before its until")
	ErrBeforeReset   = errors.New("the map was reset after this window ended: the map it played on is gone")
)

type Ledger interface {
	Replay(see func(on clicks.Stage, event ledger.Event)) ledger.Position
}

// Eras names the board the world plays, and the eras a window covers.
type Eras interface {
	Latest(ctx context.Context, played clicks.MapID) (eras.Era, error)
	Covering(ctx context.Context, played clicks.MapID, since, until time.Time) ([]eras.Era, error)
}

type Resets interface {
	After(ctx context.Context, at time.Time) ([]resets.Reset, error)
}

type Arenas interface {
	Stretches(ctx context.Context) ([]arenas.Stretch, error)
}

// Tiles clamps each range to the map's last tile.
type Tiles interface {
	Stage() clicks.Stage
	StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error)
	Kept(ctx context.Context, kept clicks.Stage) (clicks.DenseBatch, error)
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
	// The board the world plays, whose events the replay films: the opening's tiles are its tiles.
	Board eras.Board
	// The world's eras the window covers, oldest first: each names the board its tiles are on.
	Eras []eras.Era
}

func New(ledger Ledger, resets Resets, arenas Arenas, eras Eras, tiles Tiles, clock cptime.Clock) *UseCase {
	return &UseCase{ledger: ledger, resets: resets, arenas: arenas, eras: eras, tiles: tiles, clock: clock}
}

type UseCase struct {
	ledger Ledger
	resets Resets
	arenas Arenas
	eras   Eras
	tiles  Tiles
	clock  cptime.Clock
}

// Execute rewinds the world's map from now: a reset in the window starts the replay at it, on the blank map, and a
// window that ended before a reset cannot be rewound to. Only the events of the world's map on the board it plays are
// its scenes, as each event names them: an arena's are left out, the world's map came back.
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
	world, err := u.eras.Latest(ctx, clicks.World)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the board the world plays: %w", err)
	}
	covering, err := u.eras.Covering(ctx, clicks.World, in.Since, until)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the world's eras: %w", err)
	}

	// The map before the ledger: a change made between the two reads is then in the ledger, and rewound.
	now, err := u.world(ctx, world.Stage())
	if err != nil {
		return Out{}, err
	}

	footage := ledger.NewFootage(in.Since, until, world.Stage())
	u.ledger.Replay(footage.See)

	return Out{
		Since: in.Since, Until: until, Opening: footage.Opening(now), Scenes: footage.Scenes(),
		Asides: maps.Asides(in.Since, until), Board: world.Board, Eras: covering,
	}, nil
}

// The world's map as it plays, or as postgres keeps it on the board of its era while an arena's plays.
func (u *UseCase) world(ctx context.Context, world clicks.Stage) (clicks.DenseBatch, error) {
	if u.tiles.Stage().Map != clicks.World {
		kept, err := u.tiles.Kept(ctx, world)
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
