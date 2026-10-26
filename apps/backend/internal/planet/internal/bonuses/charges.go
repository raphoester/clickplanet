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

func NewCharges(store Store, rules InForce) Charges {
	return Charges{store: store, rules: rules}
}

type Charges struct {
	store Store
	rules InForce
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

	rules := c.rules.Bonuses()
	if err := c.store.Change(ctx, holder, func(held Held) (Held, error) {
		return held.Granted(kind, amount, rules), nil
	}); err != nil {
		return fmt.Errorf("failed to grant %s %d %s: %w", holder, amount, kind, err)
	}
	return nil
}

func (c Charges) Spend(ctx context.Context, holder Holder, after func(Held) (Held, bool)) error {
	if holder == NoHolder {
		return ErrNotHeld
	}

	if err := c.store.Change(ctx, holder, func(held Held) (Held, error) {
		spent, ok := after(held)
		if !ok {
			return held, ErrNotHeld
		}
		return spent, nil
	}); err != nil {
		return fmt.Errorf("failed to spend a charge of %s: %w", holder, err)
	}
	return nil
}
