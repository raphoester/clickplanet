package rallies

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// ErrUnread is a read of the rallies on that failed: the click costs a whole one.
var ErrUnread = errors.New("the rallies on could not be read")

const (
	wholeClick  = 1.0
	readTimeout = time.Second
)

type Rallying interface {
	On(ctx context.Context, flag string, at time.Time) ([]Rally, error)
}

func NewDiscount(rallying Rallying, ground Ground, rules func() charges.Rules, clock cptime.Clock) Discount {
	return Discount{rallying: rallying, ground: ground, rules: rules, clock: clock}
}

// Discount is what a click takes from the bank: charges.rally.cost on a tile inside a ring of its flag still on, a
// whole click elsewhere. Rings that overlap cost no less than one: the cost is the rules', never multiplied.
type Discount struct {
	rallying Rallying
	ground   Ground
	rules    func() charges.Rules
	clock    cptime.Clock
}

// Cost answers a whole click with ErrUnread when the rallies cannot be read: the click goes on, at full price.
func (d Discount) Cost(ctx context.Context, flag string, tile uint32) (float64, error) {
	read, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	now := d.clock.Now()
	on, err := d.rallying.On(read, flag, now)
	if err != nil {
		return wholeClick, fmt.Errorf("%w: flag %q: %w", ErrUnread, flag, err)
	}

	for _, rally := range on {
		if rally.On(now) && rally.Covers(tile, d.ground) {
			return d.rules().Rally.Cost, nil
		}
	}

	return wholeClick, nil
}
