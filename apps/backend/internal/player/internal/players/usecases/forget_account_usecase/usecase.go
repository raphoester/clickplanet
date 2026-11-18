package forget_account_usecase

import (
	"context"
	"fmt"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Accounts interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

// The operator's renames of the account, and what it did as an operator.
type Renames interface {
	Forget(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	accounts      Accounts
	renames       Renames
	outbox        cpatomic.Outbox
}

func New(transactioner cpatomic.Transactioner, accounts Accounts, renames Renames, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{transactioner: transactioner, accounts: accounts, renames: renames, outbox: outbox}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step names what failed.
		if err := u.accounts.DeleteAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to forget the account: %w", err)
		}
		if err := u.renames.Forget(ctx, account); err != nil {
			return fmt.Errorf("failed to forget the account's renames: %w", err)
		}
		if err := u.outbox.Record(ctx, &playerv1.PlayerForgotten{AccountId: account.String()}); err != nil {
			return fmt.Errorf("failed to record the account forgotten: %w", err)
		}
		return nil
	})
}
