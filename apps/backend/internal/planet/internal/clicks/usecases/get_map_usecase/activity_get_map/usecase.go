// Package activity_get_map records each map batch a caller asks the origin for, answered or refused.
package activity_get_map

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/get_map_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type UseCase interface {
	Execute(ctx context.Context, in get_map_usecase.In) (clicks.DenseBatch, error)
}

type Recorder interface {
	Record(event activity.Event)
}

type MaxIndexReader interface {
	MaxIndex() uint32
}

func New(implementation UseCase, recorder Recorder, board MaxIndexReader, clock cptime.Clock) *Decorator {
	return &Decorator{implementation: implementation, recorder: recorder, board: board, clock: clock}
}

type Decorator struct {
	implementation UseCase
	recorder       Recorder
	board          MaxIndexReader
	clock          cptime.Clock
}

func (d *Decorator) Execute(ctx context.Context, in get_map_usecase.In) (clicks.DenseBatch, error) {
	at := d.clock.Now()

	batch, err := d.implementation.Execute(ctx, in)

	d.recorder.Record(activity.Event{
		At:      at,
		Kind:    activity.KindMap,
		Caller:  activity.CallerOf(clicks.PayerOf(ctx)),
		Start:   in.Start,
		End:     in.End,
		OffMap:  in.OffMap(d.board.MaxIndex()),
		Outcome: outcomeOf(err),
	})

	return batch, err
}

func outcomeOf(err error) activity.Outcome {
	switch {
	case err == nil:
		return activity.OutcomeAccepted
	case errors.Is(err, clicks.ErrInvalidTileRange):
		return activity.OutcomeInvalid
	default:
		return activity.OutcomeFailed
	}
}
