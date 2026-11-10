package forget_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/verdicts"
)

type Store interface {
	Forget(ctx context.Context, account verdicts.AccountID) error
}

func New(store Store) *UseCase {
	return &UseCase{store: store}
}

type UseCase struct {
	store Store
}

func (u *UseCase) Execute(ctx context.Context, account verdicts.AccountID) error {
	if err := u.store.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the verdict of the account's name: %w", err)
	}
	return nil
}
