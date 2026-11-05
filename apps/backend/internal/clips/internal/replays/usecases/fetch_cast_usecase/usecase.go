package fetch_cast_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

type Executor interface {
	Execute(ctx context.Context, window replays.Window) (replays.Cast, error)
}

func New(source replays.CastSource, store replays.CastStore) *UseCase {
	return &UseCase{source: source, store: store}
}

type UseCase struct {
	source replays.CastSource
	store  replays.CastStore
}

var _ Executor = (*UseCase)(nil)

// window: the replay's, so every take of the cast is one the replay plays.
func (u *UseCase) Execute(ctx context.Context, window replays.Window) (replays.Cast, error) {
	cast, err := u.source.Cast(ctx, window)
	if err != nil {
		return replays.Cast{}, fmt.Errorf("failed to get the cast from the backend: %w", err)
	}
	if err := u.store.SaveCast(ctx, cast); err != nil {
		return replays.Cast{}, fmt.Errorf("failed to keep the cast: %w", err)
	}
	return cast, nil
}
