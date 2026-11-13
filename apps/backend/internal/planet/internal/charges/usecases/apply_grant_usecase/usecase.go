package apply_grant_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Grants interface {
	Record(ctx context.Context, grant charges.GrantID, at time.Time) (bool, error)
}

type Charger interface {
	Grant(ctx context.Context, holder charges.Holder, kind charges.Kind, amount int) (charges.Gain, error)
}

type Streams interface {
	PublishGranted(holder charges.Holder, gain charges.Gain)
}

type In struct {
	Grant   charges.GrantID
	Account string
	Kind    charges.Kind
	Amount  int
}

func New(
	transactioner cpatomic.Transactioner, grants Grants, charger Charger, streams Streams, clock cptime.Clock,
) *UseCase {
	return &UseCase{transactioner: transactioner, grants: grants, charger: charger, streams: streams, clock: clock}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	grants        Grants
	charger       Charger
	streams       Streams
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	holder, err := charges.ParseHolder(in.Account)
	if err != nil {
		return fmt.Errorf("cannot grant: %w", err)
	}
	if err := charges.GrantOf(in.Kind, in.Amount).GrantError(); err != nil {
		return fmt.Errorf("cannot grant: %w", err)
	}

	var granted *charges.Gain
	if err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		first, err := u.grants.Record(ctx, in.Grant, u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the store named it.
		}
		if !first {
			return nil
		}
		gain, err := u.charger.Grant(ctx, holder, in.Kind, in.Amount)
		if err != nil {
			return err //nolint:wrapcheck // the charger named it.
		}
		granted = &gain
		return nil
	}); err != nil {
		return err
	}

	// After the commit, so a stream never shows what a rollback took back.
	if granted != nil {
		u.streams.PublishGranted(holder, *granted)
	}
	return nil
}
