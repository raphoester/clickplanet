package fill_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Grants interface {
	Record(ctx context.Context, grant charges.GrantID, at time.Time) (bool, error)
}

type Charger interface {
	Fill(ctx context.Context, holder charges.Holder, kinds []charges.Kind) (charges.Gain, error)
}

type Streams interface {
	PublishGranted(holder charges.Holder, gain charges.Gain)
}

type In struct {
	Grant  charges.GrantID
	Holder charges.Holder
	Kinds  []charges.Kind
}

func New(
	transactioner cpatomic.Transactioner, grants Grants, charger Charger, streams Streams, clock cptime.Clock,
) *UseCase {
	return &UseCase{transactioner: transactioner, grants: grants, charger: charger, streams: streams, clock: clock}
}

// UseCase fills the stacks another module names (InternalService/FillCharges), once per grant.
type UseCase struct {
	transactioner cpatomic.Transactioner
	grants        Grants
	charger       Charger
	streams       Streams
	clock         cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	var filled *charges.Gain
	if err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		first, err := u.grants.Record(ctx, in.Grant, u.clock.Now())
		if err != nil || !first {
			return err //nolint:wrapcheck // the store named it.
		}
		gain, err := u.charger.Fill(ctx, in.Holder, in.Kinds)
		filled = &gain
		return err //nolint:wrapcheck // the charger named it.
	}); err != nil {
		return err //nolint:wrapcheck // the transaction named it.
	}

	// After the commit, so a stream never shows what a rollback took back.
	if filled != nil {
		u.streams.PublishGranted(in.Holder, *filled)
	}
	return nil
}
