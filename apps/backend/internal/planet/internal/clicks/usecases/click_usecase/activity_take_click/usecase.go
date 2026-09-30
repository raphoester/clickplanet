// Package activity_take_click records each click that changed its tile, and who held it before.
package activity_take_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Recorder interface {
	Record(event activity.Event)
}

// TileOwner is read before the write: afterwards the map no longer remembers who held the tile.
type TileOwner interface {
	Owner(tile uint32) (string, bool)
}

func New(implementation click_usecase.IUseCase, recorder Recorder, owner TileOwner, clock cptime.Clock) *UseCase {
	return &UseCase{implementation: implementation, recorder: recorder, owner: owner, clock: clock}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	recorder       Recorder
	owner          TileOwner
	clock          cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	at := u.clock.Now()
	held, known := u.owner.Owner(in.TileID)

	out, err := u.implementation.Execute(ctx, in)
	if err != nil || !known || held == in.CountryID {
		return out, err
	}

	u.recorder.Record(activity.Event{
		At:      at,
		Kind:    activity.KindTake,
		Caller:  activity.CallerOf(clicks.PayerOf(ctx)),
		Tile:    in.TileID,
		Country: in.CountryID,
		Held:    held,
	})

	return out, nil
}
