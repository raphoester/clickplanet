// Package delete_account_usecase deletes the caller's account, its identities and its sessions.
package delete_account_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	accounts.SessionFinder
	DeleteAccount(ctx context.Context, account accounts.AccountID) error
}

// Publisher is the event bus.
type Publisher interface {
	Publish(event proto.Message)
}

type UseCase struct {
	store  Store
	events Publisher
	clock  cptime.Clock
}

func New(store Store, events Publisher, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, events: events, clock: clock}
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

	// After the rows are gone: a subscriber that hears it forgets what it keeps for the account.
	u.events.Publish(&authv1.AccountDeleted{AccountId: session.Account.String()})

	return &Out{Account: session.Account, SetCookie: accounts.ExpiredSessionCookie()}, nil
}
