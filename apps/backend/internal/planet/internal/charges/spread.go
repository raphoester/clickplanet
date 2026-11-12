package charges

import (
	"context"
	"errors"
)

type Spread struct {
	Rings int
	Spent bool
}

// Taken before the click is played, so the reach and the spend agree on it.
func (c Charges) TakeSpread(ctx context.Context, holder Holder) (Spread, error) {
	rules := c.rules().Spread
	if rules.Unlimited {
		return Spread{Rings: rules.Rings}, nil
	}

	err := c.Spend(ctx, holder, Held.AfterSpreadClick)
	if errors.Is(err, ErrNotHeld) {
		return Spread{}, nil
	}
	if err != nil {
		return Spread{}, err
	}

	return Spread{Rings: rules.Rings, Spent: true}, nil
}

func (c Charges) GiveBackSpread(ctx context.Context, holder Holder, spread Spread) error {
	if !spread.Spent {
		return nil
	}

	return c.Return(ctx, holder, Held.WithSpreadClickBack)
}
