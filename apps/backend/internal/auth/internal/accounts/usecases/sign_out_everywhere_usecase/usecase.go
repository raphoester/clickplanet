// Package sign_out_everywhere_usecase ends every session of the caller's account.
package sign_out_everywhere_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	accounts.SessionFinder
	DeleteSessions(ctx context.Context, account accounts.AccountID) error
}

type UseCase struct {
	store Store
	clock cptime.Clock
}

func New(store Store, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
}

// Execute answers the Set-Cookie that clears this browser's session, or accounts.ErrNoAccount.
func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (string, error) {
	session, err := accounts.Caller(ctx, u.store, cookieHeader, u.clock.Now())
	if err != nil {
		return "", fmt.Errorf("failed to find the caller: %w", err)
	}

	if err := u.store.DeleteSessions(ctx, session.Account); err != nil {
		return "", fmt.Errorf("failed to delete the account's sessions: %w", err)
	}
	return accounts.ExpiredSessionCookie(), nil
}
