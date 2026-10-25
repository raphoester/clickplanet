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
	DeleteAccount(ctx context.Context, account accounts.AccountID, told proto.Message) error
}

type UseCase struct {
	store Store
	clock cptime.Clock
}

func New(store Store, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
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

	if err := u.store.DeleteAccount(ctx, session.Account(), &authv1.AccountDeleted{AccountId: session.Account().String()}); err != nil {
		return nil, fmt.Errorf("failed to delete the account: %w", err)
	}

	return &Out{Account: session.Account(), SetCookie: accounts.ExpiredSessionCookie()}, nil
}
