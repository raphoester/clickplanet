package apply_grant_usecase

import (
	"context"
	"fmt"
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

type In struct {
	Grant   bonuses.GrantID
	Account string
	Kind    bonuses.Kind
	Amount  int
}

func New(transactioner cpatomic.Transactioner, grants Grants, charger Charger, clock cptime.Clock) *UseCase {
	return &UseCase{transactioner: transactioner, grants: grants, charger: charger, clock: clock}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	grants        Grants
	charger       Charger
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	holder, err := bonuses.ParseHolder(in.Account)
	if err != nil {
		return fmt.Errorf("cannot grant: %w", err)
	}
	if err := bonuses.GrantOf(in.Kind, in.Amount).GrantError(); err != nil {
		return fmt.Errorf("cannot grant: %w", err)
	}

	return u.transactioner.InTx(ctx, func(ctx context.Context) error {
		first, err := u.grants.Record(ctx, in.Grant, u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the store named it.
		}
		if !first {
			return nil
		}
		return u.charger.Grant(ctx, holder, in.Kind, in.Amount) //nolint:wrapcheck // the charger named it.
	})
}
