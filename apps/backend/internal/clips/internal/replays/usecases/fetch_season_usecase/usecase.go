package fetch_season_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

type Executor interface {
	Execute(ctx context.Context) (replays.Season, error)
}

func New(source replays.SeasonSource, store replays.SeasonStore) *UseCase {
	return &UseCase{source: source, store: store}
}

type UseCase struct {
	source replays.SeasonSource
	store  replays.SeasonStore
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (replays.Season, error) {
	season, err := u.source.Season(ctx)
	if err != nil {
		return replays.Season{}, fmt.Errorf("failed to get the season from the backend: %w", err)
	}
	if err := u.store.SaveSeason(ctx, season); err != nil {
		return replays.Season{}, fmt.Errorf("failed to keep the season: %w", err)
	}
	return season, nil
}
