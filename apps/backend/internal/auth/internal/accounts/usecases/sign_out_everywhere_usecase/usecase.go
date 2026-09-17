// Package sign_out_everywhere_usecase ends every session of the caller's account.
package sign_out_everywhere_usecase

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
	DeleteSessions(ctx context.Context, account accounts.AccountID) error
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

// Execute answers the Set-Cookie that clears this browser's session, or accounts.ErrNoAccount.
func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (string, error) {
	session, err := accounts.Caller(ctx, u.store, cookieHeader, u.clock.Now())
	if err != nil {
		return "", fmt.Errorf("failed to find the caller: %w", err)
	}

	if err := u.store.DeleteSessions(ctx, session.Account); err != nil {
		return "", fmt.Errorf("failed to delete the account's sessions: %w", err)
	}

	u.events.Publish(&authv1.SignedOut{AccountId: session.Account.String()})
	return accounts.ExpiredSessionCookie(), nil
}
