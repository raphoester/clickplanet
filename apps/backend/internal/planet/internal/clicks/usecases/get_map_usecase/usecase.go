package get_map_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

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

// OffMap is a bound the web app never asks for: ids run from 1 and it clamps its last batch.
func (in In) OffMap(maxIndex uint32) bool {
	end := in.End
	if end == 0 {
		end = maxIndex
	}

	return in.Start == 0 || end > maxIndex
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
