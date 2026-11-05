package delete_account_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	accounts.SessionFinder
	accounts.AccountFinder
	RequestDeletion(ctx context.Context, account accounts.AccountID, at time.Time) error
	DeleteSessions(ctx context.Context, account accounts.AccountID) error
	DeleteAccount(ctx context.Context, account accounts.AccountID) error
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	store         Store
	outbox        cpatomic.Outbox
	clock         cptime.Clock
}

func New(transactioner cpatomic.Transactioner, store Store, outbox cpatomic.Outbox, clock cptime.Clock) *UseCase {
	return &UseCase{transactioner: transactioner, store: store, outbox: outbox, clock: clock}
}

type Out struct {
	Account   accounts.AccountID
	SetCookie string
}

// A signed-in account is signed out everywhere and kept for the grace period, so a sign-in can bring it back; the
// purge deletes it after. A guest has nothing to sign in with again, so it is deleted at once.
func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (*Out, error) {
	now := u.clock.Now()
	session, err := accounts.Caller(ctx, u.store, cookieHeader, now)
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller: %w", err)
	}
	account, err := u.store.Account(ctx, session.Account())
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller's account: %w", err)
	}

	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if !account.Restorable() {
			return u.delete(ctx, account.ID())
		}
		return u.requestDeletion(ctx, account.ID(), now)
	})
	if err != nil {
		return nil, err
	}
	return &Out{Account: account.ID(), SetCookie: accounts.ExpiredSessionCookie()}, nil
}

func (u *UseCase) requestDeletion(ctx context.Context, account accounts.AccountID, now time.Time) error {
	if err := u.store.RequestDeletion(ctx, account, now); err != nil {
		return fmt.Errorf("failed to request the account's deletion: %w", err)
	}
	if err := u.store.DeleteSessions(ctx, account); err != nil {
		return fmt.Errorf("failed to sign the account out everywhere: %w", err)
	}
	requested := &authv1.AccountDeletionRequested{AccountId: account.String(), RequestedAt: timestamppb.New(now)}
	if err := u.outbox.Record(ctx, requested); err != nil {
		return fmt.Errorf("failed to record the deletion request: %w", err)
	}
	return nil
}

func (u *UseCase) delete(ctx context.Context, account accounts.AccountID) error {
	if err := u.store.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to delete the account: %w", err)
	}
	if err := u.outbox.Record(ctx, &authv1.AccountDeleted{AccountId: account.String()}); err != nil {
		return fmt.Errorf("failed to record the deletion: %w", err)
	}
	return nil
}
