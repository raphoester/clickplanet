package tell_sign_in_usecase

import (
	"context"
	"fmt"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type UseCase struct {
	transactioner cpatomic.Transactioner
	outbox        cpatomic.Outbox
}

func New(transactioner cpatomic.Transactioner, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{transactioner: transactioner, outbox: outbox}
}

// Every sign-in, whether or not player names the account: a module that waits for one must not wait on a name drawn.
func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	return u.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // the step names what failed.
		if err := u.outbox.Record(ctx, &playerv1.PlayerSignedIn{AccountId: account.String()}); err != nil {
			return fmt.Errorf("failed to record the sign-in: %w", err)
		}
		return nil
	})
}
