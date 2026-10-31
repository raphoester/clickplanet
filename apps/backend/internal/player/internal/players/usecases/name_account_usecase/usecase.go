package name_account_usecase

import (
	"context"
	"fmt"
	"time"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Names interface {
	Assign(ctx context.Context, account players.AccountID, suggested string, at time.Time) (bool, error)
}

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	names         Names
	profiles      Profiles
	outbox        cpatomic.Outbox
	clock         cptime.Clock
}

func New(
	transactioner cpatomic.Transactioner, names Names, profiles Profiles, outbox cpatomic.Outbox, clock cptime.Clock,
) *UseCase {
	return &UseCase{transactioner: transactioner, names: names, profiles: profiles, outbox: outbox, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID, suggested string) (players.Profile, error) {
	err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		created, err := u.names.Assign(ctx, account, suggested, u.clock.Now())
		if err != nil {
			return fmt.Errorf("failed to name the account: %w", err)
		}
		if !created {
			return nil
		}
		if err := u.outbox.Record(ctx, &playerv1.PlayerCreated{AccountId: account.String()}); err != nil {
			return fmt.Errorf("failed to record the player made: %w", err)
		}
		return nil
	})
	if err != nil {
		return players.Profile{}, err //nolint:wrapcheck // the transaction's step named it.
	}

	profile, err := u.profiles.Profile(ctx, account)
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the account's name: %w", err)
	}
	return profile, nil
}
