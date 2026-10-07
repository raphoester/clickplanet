package replay_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var ErrInvalidWindow = errors.New("a replay needs a since before its until")

type Ledger interface {
	Replay(see func(ledger.Event)) ledger.Position
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

func New(ledger Ledger, board MaxIndexReader, tiles DenseMapReader, clock cptime.Clock) *UseCase {
	return &UseCase{ledger: ledger, board: board, tiles: tiles, clock: clock}
}

type UseCase struct {
	ledger Ledger
	board  MaxIndexReader
	tiles  DenseMapReader
	clock  cptime.Clock
}

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	until := in.Until
	if until.IsZero() {
		until = u.clock.Now()
	}
	if in.Since.IsZero() || !in.Since.Before(until) {
		return Out{}, ErrInvalidWindow
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
