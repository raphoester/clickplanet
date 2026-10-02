package clicks

import (
	"context"
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type TollStep struct {
	Share    float64
	Slowdown float64
}

type TollConfig struct {
	Steps []TollStep
}

func (c TollConfig) Validate() error {
	previous := TollStep{Slowdown: 1}

	for i, step := range c.Steps {
		if math.IsNaN(step.Share) || step.Share <= previous.Share || step.Share > 1 {
			return fmt.Errorf("toll.steps[%d].share is %v: shares must rise, above 0 and up to 1", i, step.Share)
		}
		if math.IsNaN(step.Slowdown) || math.IsInf(step.Slowdown, 0) || step.Slowdown <= previous.Slowdown {
			return fmt.Errorf("toll.steps[%d].slowdown is %v: slowdowns must rise, above 1", i, step.Slowdown)
		}
		previous = step
	}

	return nil
}

type Price struct {
	// The country priced: the payer's main flag, which is not always the one it clicks for now.
	Country string

	Slowdown float64
	Share    float64

	NextShare    float64
	NextSlowdown float64
}

type Budget struct {
	cpratelimit.State
	SharedWith       SharedWith
	Price            Price
	LinkedMultiplier float64
}

type ShareReader interface {
	Share(country string) float64
}

// Flags is every tally of who takes tiles for which flag, by key. A key with no tally is absent.
type Flags interface {
	Allegiances(ctx context.Context, keys ...AllegianceKey) (map[AllegianceKey]Allegiance, error)
}

func NewToll(config TollConfig, shares ShareReader, flags Flags, clock cptime.Clock) *Toll {
	return &Toll{steps: config.Steps, shares: shares, flags: flags, clock: clock}
}

type Toll struct {
	steps  []TollStep
	shares ShareReader
	flags  Flags
	clock  cptime.Clock
}

// PriceFor is the price of payer's next click for country: its own tally's main flag once that click counts.
func (t *Toll) PriceFor(ctx context.Context, payer Payer, country string) (Price, error) {
	key := payer.AllegianceKey()

	tallies, err := t.flags.Allegiances(ctx, key)
	if err != nil {
		return Price{}, fmt.Errorf("failed to read the payer's flag: %w", err)
	}

	return t.Price(tallies[key].With(country, t.clock.Now()).Flag()), nil
}

func (t *Toll) Price(country string) Price {
	price := Price{Country: country, Slowdown: 1, Share: t.shares.Share(country)}

	for _, step := range t.steps {
		if price.Share < step.Share {
			price.NextShare, price.NextSlowdown = step.Share, step.Slowdown
			break
		}
		price.Slowdown = step.Slowdown
	}

	return price
}
