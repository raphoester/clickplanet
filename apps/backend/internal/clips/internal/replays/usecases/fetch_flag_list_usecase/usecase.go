package fetch_flag_list_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/replays"
)

type Executor interface {
	Execute(ctx context.Context) (replays.FlagList, error)
}

func New(source replays.FlagListSource, store replays.FlagListStore) *UseCase {
	return &UseCase{source: source, store: store}
}

type UseCase struct {
	source replays.FlagListSource
	store  replays.FlagListStore
}

var _ Executor = (*UseCase)(nil)

func (u *UseCase) Execute(ctx context.Context) (replays.FlagList, error) {
	list, err := u.source.FlagList(ctx)
	if err != nil {
		return replays.FlagList{}, fmt.Errorf("failed to get the flag list from the backend: %w", err)
	}
	if err := u.store.SaveFlagList(ctx, list); err != nil {
		return replays.FlagList{}, fmt.Errorf("failed to keep the flag list: %w", err)
	}
	return list, nil
}
