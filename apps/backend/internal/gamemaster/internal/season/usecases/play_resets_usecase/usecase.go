package play_resets_usecase

import (
	"context"
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Kept interface {
	KeepReset(ctx context.Context, book moments.BookID) (bool, error)
}

type Planet interface {
	// Reset asks planet to start the season on a blank map: nil once it is wiped, moments.ErrRefused for a refusal for
	// good, any other error no answer.
	Reset(ctx context.Context, starts season.Window) error
}

// Execute answers the season whose reset it asked, and done is true when planet answered it wiped.
type Executor interface {
	Execute(ctx context.Context, plan moments.Plan) (starts season.Window, done bool, err error)
}

func New(kept Kept, planet Planet, clock cptime.Clock, again time.Duration) *UseCase {
	return &UseCase{kept: kept, planet: planet, clock: clock, calls: moments.NewCalls[season.Number](again)}
}

// UseCase asks planet to reset the map of the season that runs once the plan says it may be wiped (Plan.ResetAt), until
// planet answers, and keeps the answer right after the call. A call planet did not answer is made again after `again`;
// one it refused for good is not made again in this process.
type UseCase struct {
	kept   Kept
	planet Planet
	clock  cptime.Clock

	calls *moments.Calls[season.Number]
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, plan moments.Plan) (season.Window, bool, error) {
	now := u.clock.Now()
	running, ok := plan.ResetAt(now)
	if !ok || u.calls.Answered(running.Number) || u.calls.Waits(running.Number, now) {
		return season.Window{}, false, nil
	}
	err := u.call(ctx, running)
	switch {
	case errors.Is(err, moments.ErrRefused):
		u.calls.Answer(running.Number)
		return running, false, err
	case err != nil:
		u.calls.Fail(running.Number, now)
		return running, false, err
	}
	u.calls.Answer(running.Number)
	return running, true, nil
}

func (u *UseCase) call(ctx context.Context, running season.Window) error {
	if err := u.planet.Reset(ctx, running); err != nil {
		return err //nolint:wrapcheck // the adapter names the season.
	}
	_, err := u.kept.KeepReset(ctx, moments.BookOf(running.Number))
	return err //nolint:wrapcheck // the store names the season.
}
