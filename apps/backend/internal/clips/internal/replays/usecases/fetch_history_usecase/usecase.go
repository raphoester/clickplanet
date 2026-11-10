package fetch_history_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

type Executor interface {
	Execute(ctx context.Context, turn replays.Turn, until time.Time) (replays.History, error)
}

func New(source replays.HistorySource, store replays.HistoryStore) *UseCase {
	return &UseCase{source: source, store: store}
}

type UseCase struct {
	source replays.HistorySource
	store  replays.HistoryStore
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context, turn replays.Turn, until time.Time) (replays.History, error) {
	window, err := turn.SeasonUntil(until)
	if err != nil {
		return replays.History{}, fmt.Errorf("failed to ask for the season's map: %w", err)
	}
	history, err := u.source.History(ctx, window)
	if err != nil {
		return replays.History{}, fmt.Errorf("failed to get the season's map from the backend: %w", err)
	}
	if err := u.store.SaveHistory(ctx, history); err != nil {
		return replays.History{}, fmt.Errorf("failed to keep the season's map: %w", err)
	}
	return history, nil
}
