package fetch_round_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

type Executor interface {
	Execute(ctx context.Context, turn replays.Turn) (replays.Round, error)
}

func New(source replays.RoundSource, store replays.RoundStore) *UseCase {
	return &UseCase{source: source, store: store}
}

type UseCase struct {
	source replays.RoundSource
	store  replays.RoundStore
}

var _ Executor = (*UseCase)(nil)

// A turn with no debrief keeps an empty round, so a round of an earlier run is never told again.
func (u *UseCase) Execute(ctx context.Context, turn replays.Turn) (replays.Round, error) {
	round := replays.NoRound
	if turn.Round != "" {
		var err error
		if round, err = u.source.Round(ctx, turn.Round); err != nil {
			return replays.Round{}, fmt.Errorf("failed to get round %q from the backend: %w", turn.Round, err)
		}
	}
	if err := u.store.SaveRound(ctx, round); err != nil {
		return replays.Round{}, fmt.Errorf("failed to keep the round: %w", err)
	}
	return round, nil
}
