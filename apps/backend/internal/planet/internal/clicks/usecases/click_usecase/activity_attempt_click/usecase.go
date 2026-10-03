package activity_attempt_click

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/activity"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Recorder interface {
	Record(event activity.Event)
}

// TileOwner decides a no-op, not Out.Outcome: a dropped click answers Unchanged, which would leak the ban.
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

	u.recorder.Record(activity.Event{
		At:      at,
		Kind:    activity.KindClick,
		Caller:  activity.CallerOf(clicks.PayerOf(ctx)),
		Tile:    in.TileID,
		Country: in.CountryID,
		Outcome: outcomeOf(err, known && held == in.CountryID),
	})

	return out, err
}

// invalid is what click_handler answers 400 for.
var invalid = []error{clicks.ErrUnknownCountry, clicks.ErrTileOutOfRange, clicks.ErrBonusesTogether}

func outcomeOf(err error, noOp bool) activity.Outcome {
	switch {
	case err == nil && noOp:
		return activity.OutcomeNoOp
	case err == nil:
		return activity.OutcomeAccepted
	case errors.Is(err, clicks.ErrThrottled):
		return activity.OutcomeThrottled
	}

	for _, refusal := range invalid {
		if errors.Is(err, refusal) {
			return activity.OutcomeInvalid
		}
	}

	return activity.OutcomeFailed
}
