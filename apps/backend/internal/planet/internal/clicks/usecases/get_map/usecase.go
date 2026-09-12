// Package get_map answers a range of the tile map as one dense batch.
package get_map

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// MaxIndexReader is only consulted for the open-ended request: an unset end
// means "to the end of the map", and the map is the only thing that knows where
// that is.
type MaxIndexReader interface {
	MaxIndex() uint32
}

type DenseMapReader interface {
	StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error)
}

type In struct {
	Start uint32
	End   uint32
}

func New(tilesChecker MaxIndexReader, mapReader DenseMapReader) *UseCase {
	return &UseCase{
		tilesChecker: tilesChecker,
		mapReader:    mapReader,
	}
}

type UseCase struct {
	tilesChecker MaxIndexReader
	mapReader    DenseMapReader
}

// An inverted or out-of-range span is the caller's mistake, so the reader's
// complaint is carried out under the sentinel that says which mistake it was.
func (u *UseCase) Execute(_ context.Context, in In) (clicks.DenseBatch, error) {
	end := in.End
	if end == 0 {
		end = u.tilesChecker.MaxIndex()
	}

	batch, err := u.mapReader.StateBatchDense(in.Start, end)
	if err != nil {
		return clicks.DenseBatch{}, fmt.Errorf("%w: %w", clicks.ErrInvalidTileRange, err)
	}

	return batch, nil
}
