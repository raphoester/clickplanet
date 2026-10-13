package fetch_replay_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Executor interface {
	Execute(ctx context.Context) (replays.Replay, error)
}

func New(asked replays.Request, clock cptime.Clock, source replays.Source, store replays.Store) *UseCase {
	return &UseCase{asked: asked, clock: clock, source: source, store: store}
}

type UseCase struct {
	asked  replays.Request
	clock  cptime.Clock
	source replays.Source
	store  replays.Store
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (replays.Replay, error) {
	window, err := u.asked.Window(u.clock.Now())
	if err != nil {
		return replays.Replay{}, err //nolint:wrapcheck // a pure delegation: the request already named what is wrong.
	}

	replay, err := u.source.Replay(ctx, window)
	if err != nil {
		return replays.Replay{}, fmt.Errorf("failed to get the replay from the backend: %w", err)
	}
	if err := u.store.Save(ctx, replay); err != nil {
		return replays.Replay{}, fmt.Errorf("failed to keep the replay: %w", err)
	}
	return replay, nil
}
