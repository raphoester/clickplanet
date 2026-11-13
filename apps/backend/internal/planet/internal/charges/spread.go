package charges

import (
	"context"
	"errors"
)

// TakeSpread answers how many rings an accepted click with spread on spreads: 0 when no spread click is held.
func (c Charges) TakeSpread(ctx context.Context, holder Holder) (int, error) {
	rules := c.rules().Spread
	if rules.Unlimited {
		return rules.Rings, nil
	}

	err := c.Spend(ctx, holder, Held.AfterSpreadClick)
	if errors.Is(err, ErrNotHeld) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return rules.Rings, nil
}
