// Package delete_account_usecase deletes the caller's account, its identities and its sessions.
package delete_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	accounts.SessionFinder
	DeleteAccount(ctx context.Context, account accounts.AccountID) error
}

type UseCase struct {
	store Store
	clock cptime.Clock
}

func New(store Store, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
}

// Out is the account that is gone, and the Set-Cookie that clears the session.
type Out struct {
	Account   accounts.AccountID
	SetCookie string
}

// Execute answers accounts.ErrNoAccount when the cookie holds no live session.
func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (*Out, error) {
	session, err := accounts.Caller(ctx, u.store, cookieHeader, u.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller: %w", err)
	}

	if err := u.store.DeleteAccount(ctx, session.Account); err != nil {
		return nil, fmt.Errorf("failed to delete the account: %w", err)
	}

	// The seam for auth.v1.AccountDeleted: once the event bus exists, publish it here, after the rows are gone.
	return &Out{Account: session.Account, SetCookie: accounts.ExpiredSessionCookie()}, nil
}
