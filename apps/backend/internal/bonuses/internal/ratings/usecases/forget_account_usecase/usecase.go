package forget_account_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
)

type Store interface {
	ForgetAccount(ctx context.Context, account ratings.AccountID) error
}

func New(store Store) *UseCase {
	return &UseCase{store: store}
}

// UseCase blanks a forgotten account's id on every answer it gave. The answers stay, and so do the ratings they moved:
// nothing of them names the account any more.
type UseCase struct {
	store Store
}

func (u *UseCase) Execute(ctx context.Context, account ratings.AccountID) error {
	return u.store.ForgetAccount(ctx, account) //nolint:wrapcheck // the store named it.
}
