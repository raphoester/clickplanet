package bonuses

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotHeld = errors.New("no such charge held")

type Store interface {
	// Change runs under a lock on the holder, and keeps what it answers when that differs: an empty hand as no row.
	Change(ctx context.Context, holder Holder, change func(Held) (Held, error)) error
	Held(ctx context.Context, holder Holder) (Held, error)
}

func NewCharges(store Store, config ChargesConfig) Charges {
	return Charges{store: store, config: config}
}

type Charges struct {
	store  Store
	config ChargesConfig
}

func (c Charges) Held(ctx context.Context, holder Holder) (Held, error) {
	if holder == NoHolder {
		return Held{}, nil
	}

	held, err := c.store.Held(ctx, holder)
	if err != nil {
		return Held{}, fmt.Errorf("failed to read what %s holds: %w", holder, err)
	}

	return held, nil
}

func (c Charges) Grant(ctx context.Context, holder Holder, kind Kind, amount int) error {
	if holder == NoHolder {
		return nil
	}

	if err := c.store.Change(ctx, holder, func(held Held) (Held, error) {
		return held.Granted(kind, amount, c.config), nil
	}); err != nil {
		return fmt.Errorf("failed to grant %s %d %s: %w", holder, amount, kind, err)
	}

	return nil
}

func (c Charges) SpendRefill(ctx context.Context, holder Holder) error {
	return c.spend(ctx, holder, KindRefill, Held.AfterRefill)
}

func (c Charges) SpendBomb(ctx context.Context, holder Holder) error {
	return c.spend(ctx, holder, KindBomb, Held.AfterBomb)
}

func (c Charges) SpendEnclose(ctx context.Context, holder Holder) error {
	return c.spend(ctx, holder, KindEncloseClicks, Held.AfterEnclose)
}

func (c Charges) SpendSpreadClick(ctx context.Context, holder Holder) error {
	return c.spend(ctx, holder, KindSpreadClicks, Held.AfterSpreadClick)
}

func (c Charges) SpendShield(ctx context.Context, holder Holder) error {
	return c.spend(ctx, holder, KindShields, Held.AfterShield)
}

func (c Charges) spend(ctx context.Context, holder Holder, kind Kind, after func(Held) (Held, bool)) error {
	if holder == NoHolder {
		return fmt.Errorf("%w: no account holds a %s", ErrNotHeld, kind)
	}

	if err := c.store.Change(ctx, holder, func(held Held) (Held, error) {
		spent, ok := after(held)
		if !ok {
			return held, ErrNotHeld
		}

		return spent, nil
	}); err != nil {
		return fmt.Errorf("failed to spend %s's %s: %w", holder, kind, err)
	}

	return nil
}
