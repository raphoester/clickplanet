package fetch_race_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

type Executor interface {
	Execute(ctx context.Context) (replays.Race, error)
}

func New(source replays.RaceSource, store replays.RaceStore) *UseCase {
	return &UseCase{source: source, store: store}
}

type UseCase struct {
	source replays.RaceSource
	store  replays.RaceStore
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (replays.Race, error) {
	race, err := u.source.Race(ctx)
	if err != nil {
		return replays.Race{}, fmt.Errorf("failed to get the race from the backend: %w", err)
	}
	if err := u.store.SaveRace(ctx, race); err != nil {
		return replays.Race{}, fmt.Errorf("failed to keep the race: %w", err)
	}
	return race, nil
}
