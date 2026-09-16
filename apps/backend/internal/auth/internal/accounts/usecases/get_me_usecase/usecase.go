// Package get_me_usecase reads the account a browser's cookie belongs to. It creates, extends and saves nothing.
package get_me_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	accounts.SessionFinder
	accounts.AccountFinder
}

type UseCase struct {
	store Store
	clock cptime.Clock
}

func New(store Store, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
}

// Execute answers accounts.ErrNoAccount when the cookie holds no live session.
func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (*accounts.Account, error) {
	session, err := accounts.Caller(ctx, u.store, cookieHeader, u.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller: %w", err)
	}

	account, err := u.store.FindAccount(ctx, session.Account)
	if errors.Is(err, accounts.ErrAccountNotFound) {
		return nil, fmt.Errorf("%w: %w", accounts.ErrNoAccount, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the account: %w", err)
	}
	return account, nil
}
