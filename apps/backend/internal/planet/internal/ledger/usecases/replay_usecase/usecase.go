package replay_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

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

type MaxIndexReader interface {
	MaxIndex() uint32
}

type DenseMapReader interface {
	StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error)
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
}

func New(ledger Ledger, resets Resets, board MaxIndexReader, tiles DenseMapReader, clock cptime.Clock) *UseCase {
	return &UseCase{ledger: ledger, resets: resets, board: board, tiles: tiles, clock: clock}
}

type UseCase struct {
	ledger Ledger
	resets Resets
	board  MaxIndexReader
	tiles  DenseMapReader
	clock  cptime.Clock
}

// Execute rewinds the map from now: a reset in the window starts the replay at it, on the blank map, and a window that
// ended before a reset cannot be rewound to.
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

	// The map before the ledger: a change made between the two reads is then in the ledger, and rewound.
	now, err := u.tiles.StateBatchDense(1, u.board.MaxIndex())
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the map: %w", err)
	}

	footage := ledger.NewFootage(in.Since, until)
	u.ledger.Replay(footage.See)

	return Out{Since: in.Since, Until: until, Opening: footage.Opening(now), Scenes: footage.Scenes()}, nil
}
