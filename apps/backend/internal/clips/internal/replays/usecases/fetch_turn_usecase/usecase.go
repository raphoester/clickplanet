package fetch_turn_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

type Executor interface {
	Execute(ctx context.Context) (replays.Turn, error)
}

func New(source replays.TurnSource, store replays.TurnStore) *UseCase {
	return &UseCase{source: source, store: store}
}

type UseCase struct {
	source replays.TurnSource
	store  replays.TurnStore
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (replays.Turn, error) {
	turn, err := u.source.Turn(ctx)
	if err != nil {
		return replays.Turn{}, fmt.Errorf("failed to get the turn from the backend: %w", err)
	}
	if err := u.store.SaveTurn(ctx, turn); err != nil {
		return replays.Turn{}, fmt.Errorf("failed to keep the turn: %w", err)
	}
	return turn, nil
}
