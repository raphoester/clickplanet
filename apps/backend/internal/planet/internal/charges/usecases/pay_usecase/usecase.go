package pay_usecase

import (
	"context"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Grants keeps each grant paid, in the payment's transaction where it can: a grant kept already is told again.
type Grants interface {
	Record(ctx context.Context, grant charges.GrantID, at time.Time) (bool, error)
}

type Charger interface {
	Grant(ctx context.Context, holder charges.Holder, kind charges.Kind, amount int) (charges.Gain, error)
	Held(ctx context.Context, holder charges.Holder) (charges.Held, error)
}

// Streams tell the account's own streams what a grant added, so the page shows a reward the answer did not carry.
type Streams interface {
	PublishGranted(holder charges.Holder, gain charges.Gain)
}

type In struct {
	Grant  charges.GrantID
	Holder charges.Holder
	Kind   charges.Kind
	Amount int
}

// Out is how many of the kind the hand kept, fewer than granted when its stack is full, and the hand after.
type Out struct {
	Kept int
	Held charges.Held
}

func New(
	transactioner cpatomic.Transactioner, grants Grants, charger Charger, streams Streams, clock cptime.Clock,
) *UseCase {
	return &UseCase{transactioner: transactioner, grants: grants, charger: charger, streams: streams, clock: clock}
}

// UseCase pays a grant another module makes at once (InternalService/GrantCharges). It is the GrantCharges command's
// path (apply_grant_usecase) with an answer, since the player is told what it kept.
type UseCase struct {
	transactioner cpatomic.Transactioner
	grants        Grants
	charger       Charger
	streams       Streams
	clock         cptime.Clock
}

// Execute grants once per grant: told again, it keeps nothing and answers the hand as it is.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	var (
		out     Out
		granted *charges.Gain
	)
	if err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		first, err := u.grants.Record(ctx, in.Grant, u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the store named it.
		}
		if !first {
			held, err := u.charger.Held(ctx, in.Holder)
			out = Out{Held: held}
			return err
		}
		gain, err := u.charger.Grant(ctx, in.Holder, in.Kind, in.Amount)
		out, granted = Out{Kept: gain.Count(in.Kind), Held: gain.After}, &gain
		return err
	}); err != nil {
		return Out{}, err
	}

	// After the commit, so a stream never shows what a rollback took back.
	if granted != nil {
		u.streams.PublishGranted(in.Holder, *granted)
	}
	return out, nil
}
