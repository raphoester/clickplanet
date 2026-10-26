package get_map_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type Board interface {
	MaxIndex() uint32
	OffMap(start, end uint32) bool
}

type DenseMapReader interface {
	StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error)
}

type Facts interface {
	Tell(fact facts.Fact)
}

type In struct {
	Start uint32
	End   uint32
}

func New(board Board, mapReader DenseMapReader, facts Facts) *UseCase {
	return &UseCase{board: board, mapReader: mapReader, facts: facts}
}

type UseCase struct {
	board     Board
	mapReader DenseMapReader
	facts     Facts
}

func (u *UseCase) Execute(ctx context.Context, in In) (clicks.DenseBatch, error) {
	maxIndex := u.board.MaxIndex()

	end := in.End
	if end == 0 {
		end = maxIndex
	}

	batch, err := u.mapReader.StateBatchDense(in.Start, end)
	if err != nil {
		return clicks.DenseBatch{}, fmt.Errorf("%w: %w", clicks.ErrInvalidTileRange, err)
	}

	read := &planetv1.MapRead{
		Scope:  cpctx.RateLimitKey(ctx),
		Maps:   float64(len(batch.Tiles)/2) / float64(maxIndex),
		OffMap: u.board.OffMap(in.Start, end),
	}
	u.facts.Tell(func(at time.Time) proto.Message {
		read.ReadAt = timestamppb.New(at)
		return read
	})

	return batch, nil
}
