package grant_sign_in_refill_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Grants interface {
	Record(ctx context.Context, grant bonuses.GrantID, at time.Time) (bool, error)
}

type Charger interface {
	Grant(ctx context.Context, holder bonuses.Holder, kind bonuses.Kind, amount int) error
}

func New(
	transactioner cpatomic.Transactioner, grants Grants, charger Charger, rules func() bonuses.Rules, clock cptime.Clock,
) *UseCase {
	return &UseCase{transactioner: transactioner, grants: grants, charger: charger, rules: rules, clock: clock}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	grants        Grants
	charger       Charger
	rules         func() bonuses.Rules
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, holder bonuses.Holder) error {
	if !u.rules().SignInRefill {
		return nil
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error {
		first, err := u.grants.Record(ctx, bonuses.SignInGrantOf(holder), u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the store named it.
		}
		if !first {
			return nil
		}
		return u.charger.Grant(ctx, holder, bonuses.KindRefill, 1) //nolint:wrapcheck // the charger named it.
	})
}
