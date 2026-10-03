package activity_listen_for_events

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type UseCase interface {
	Execute(ctx context.Context, sink listen_for_events_usecase.Sink) error
}

type Recorder interface {
	Record(event activity.Event)
}

func New(implementation UseCase, recorder Recorder, clock cptime.Clock) *Decorator {
	return &Decorator{implementation: implementation, recorder: recorder, clock: clock}
}

type Decorator struct {
	implementation UseCase
	recorder       Recorder
	clock          cptime.Clock
}

func (d *Decorator) Execute(ctx context.Context, sink listen_for_events_usecase.Sink) error {
	d.recorder.Record(activity.Event{At: d.clock.Now(), Kind: activity.KindStream, Caller: activity.CallerOf(clicks.PayerOf(ctx))})

	return d.implementation.Execute(ctx, sink)
}
