package placements

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Charges interface {
	Spend(ctx context.Context, holder bonuses.Holder, after func(bonuses.Held) (bonuses.Held, bool)) error
	Return(ctx context.Context, holder bonuses.Holder, back func(bonuses.Held) bonuses.Held) error
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

func NewEscrow(transactioner cpatomic.Transactioner, charges Charges, store Store, ids IDProvider, clock cptime.Clock) Escrow {
	return Escrow{transactioner: transactioner, charges: charges, store: store, ids: ids, clock: clock}
}

// Escrow holds a shield from the hand it left until the map saves it, or until it goes back to that hand.
type Escrow struct {
	transactioner cpatomic.Transactioner
	charges       Charges
	store         Store
	ids           IDProvider
	clock         cptime.Clock
}

// Take answers what the hand holds once the shield left it.
func (e Escrow) Take(ctx context.Context, holder bonuses.Holder) (bonuses.Held, clicks.PlacementID, error) {
	id, err := e.ids.NewID()
	if err != nil {
		return bonuses.Held{}, clicks.PlacementID{}, fmt.Errorf("failed to name a placement: %w", err)
	}

	var held bonuses.Held
	if err := e.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := e.charges.Spend(ctx, holder, bonuses.Held.AfterShield); err != nil {
			return err //nolint:wrapcheck // the charges name it, and the caller reads bonuses.ErrNotHeld.
		}
		placement, err := New(id, holder, e.clock.Now())
		if err != nil {
			return err
		}
		if err := e.store.Save(ctx, placement); err != nil {
			return err //nolint:wrapcheck // the store names it.
		}
		held, err = e.charges.Held(ctx, holder)
		return err //nolint:wrapcheck // the charges name it.
	}); err != nil {
		return bonuses.Held{}, clicks.PlacementID{}, err //nolint:wrapcheck // each step names itself.
	}
	return held, id, nil
}

// Confirm ends the placements the map saved, in the transaction that saved their shields.
func (e Escrow) Confirm(ctx context.Context, ids []clicks.PlacementID) error {
	if len(ids) == 0 {
		return nil
	}

	return e.transactioner.InTx(ctx, func(ctx context.Context) error { //nolint:wrapcheck // each step names itself.
		found, err := e.store.Find(ctx, ids...)
		if err != nil {
			return err //nolint:wrapcheck // the store names it.
		}
		ended := make([]Placement, 0, len(found))
		for _, placement := range found {
			ended = append(ended, placement.Ended())
		}
		return e.store.Save(ctx, ended...) //nolint:wrapcheck // the store names it.
	})
}

// Return gives each shield back to its hand, once: a placement that ended is not found.
func (e Escrow) Return(ctx context.Context, ids ...clicks.PlacementID) error {
	errs := make([]error, 0, len(ids))
	for _, id := range ids {
		errs = append(errs, e.transactioner.InTx(ctx, func(ctx context.Context) error {
			found, err := e.store.Find(ctx, id)
			if err != nil {
				return err //nolint:wrapcheck // the store names it.
			}
			for _, placement := range found {
				if err := e.charges.Return(ctx, placement.Holder(), bonuses.Held.WithShieldBack); err != nil {
					return err //nolint:wrapcheck // the charges name it.
				}
				if err := e.store.Save(ctx, placement.Ended()); err != nil {
					return err //nolint:wrapcheck // the store names it.
				}
			}
			return nil
		}))
	}
	return errors.Join(errs...)
}
