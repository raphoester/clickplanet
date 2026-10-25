package delete_account_usecase

import (
	"context"
	"fmt"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	accounts.SessionFinder
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

func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (*Out, error) {
	session, err := accounts.Caller(ctx, u.store, cookieHeader, u.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller: %w", err)
	}

	err = u.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := u.store.DeleteAccount(ctx, session.Account()); err != nil {
			return fmt.Errorf("failed to delete the account: %w", err)
		}
		if err := u.outbox.Record(ctx, &authv1.AccountDeleted{AccountId: session.Account().String()}); err != nil {
			return fmt.Errorf("failed to record the deletion: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Out{Account: session.Account(), SetCookie: accounts.ExpiredSessionCookie()}, nil
}
