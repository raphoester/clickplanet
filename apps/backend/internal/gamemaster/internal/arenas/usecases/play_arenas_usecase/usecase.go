package play_arenas_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Kept interface {
	KeepOpened(ctx context.Context, arena moments.RoundID) (bool, error)
	KeepClosed(ctx context.Context, arena moments.RoundID) (bool, error)
}

// Planet opens and closes arenas: nil is open or closed, moments.ErrRefused a refusal for good, any other error no
// answer.
type Planet interface {
	Ask(ctx context.Context, command moments.ArenaCommand) error
}

// Call is an arena command planet answered this time, or refused for good.
type Call struct {
	Command moments.ArenaCommand
	Refusal error
}

// Execute answers each arena command planet answered this time.
type Executor interface {
	Execute(ctx context.Context, plan moments.Plan) ([]Call, error)
}

func New(kept Kept, planet Planet, clock cptime.Clock, again time.Duration) *UseCase {
	return &UseCase{kept: kept, planet: planet, clock: clock, calls: moments.NewCalls[moments.ArenaKey](again)}
}

// UseCase calls planet with each arena command due, in order, until it answers, keeping each answer right after the call. A call planet did not answer is made again after `again`, and holds back
// the ones after it.
type UseCase struct {
	kept   Kept
	planet Planet
	clock  cptime.Clock

	calls *moments.Calls[moments.ArenaKey]
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, plan moments.Plan) ([]Call, error) {
	now := u.clock.Now()
	var calls []Call
	for _, command := range plan.ArenasDueAt(now) {
		key := command.Key()
		if u.calls.Answered(key) {
			continue
		}
		if u.calls.Waits(key, now) {
			return calls, nil
		}
		err := u.call(ctx, command)
		if errors.Is(err, moments.ErrRefused) {
			u.calls.Answer(key)
			calls = append(calls, Call{Command: command, Refusal: err})
			continue
		}
		if err != nil {
			u.calls.Fail(key, now)
			return calls, err
		}
		u.calls.Answer(key)
		calls = append(calls, Call{Command: command})
	}
	return calls, nil
}

func (u *UseCase) call(ctx context.Context, command moments.ArenaCommand) error {
	if err := u.planet.Ask(ctx, command); err != nil {
		return err //nolint:wrapcheck // the adapter names the arena.
	}
	switch command.Action {
	case moments.OpenArena:
		_, err := u.kept.KeepOpened(ctx, command.Arena)
		return err //nolint:wrapcheck // the store names the arena.
	case moments.CloseArena:
		_, err := u.kept.KeepClosed(ctx, command.Arena)
		return err //nolint:wrapcheck // the store names the arena.
	}
	return fmt.Errorf("planet has no arena call to %s", command.Action)
}
