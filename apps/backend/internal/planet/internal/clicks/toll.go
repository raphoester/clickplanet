package clicks

import (
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// TollStep is one row of the table: from Share of the map, a player of that country gets its clicks back
// Slowdown times slower. Slowdown may be a fraction: 1.5 is half as slow again.
type TollStep struct {
	Share    float64
	Slowdown float64
}

// TollConfig holds the steps, lowest share first. Empty refills every country at the plain rate.
type TollConfig struct {
	Steps []TollStep
}

// Validate refuses shares or slowdowns that do not rise: a step that made a bigger country faster would be a
// reward for leading.
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

// Price is how much slower a country's players get their clicks back now, and from which share it slows next.
type Price struct {
	// The country priced: the payer's main flag, which is not always the one it clicks for now.
	Country string

	Slowdown float64
	Share    float64

	// The share at which the next step starts, and its slowdown. Zero at the top step.
	NextShare    float64
	NextSlowdown float64
}

// Budget is an allowance: every click costs one token, so it is counted in clicks. Price is the caller's main
// flag's; LinkedMultiplier is what signing in multiplies the refill by, the same for every caller.
type Budget struct {
	cpratelimit.State
	Price            Price
	LinkedMultiplier float64
}

type ShareReader interface {
	Share(country string) float64
}

// Flags is the flag each account and each scope clicks for most, as the click chain records it.
type Flags interface {
	OfAccount(account string) Allegiance
	OfScope(scope string) Allegiance
}

func NewToll(config TollConfig, shares ShareReader, flags Flags, clock cptime.Clock) *Toll {
	return &Toll{steps: config.Steps, shares: shares, flags: flags, clock: clock}
}

// A toll slows the refill of a player by how much of the map its country already holds.
//
// Every click costs one token; the price is the pace the bucket refills at afterwards, set by each click from the
// player's main flag, the one it clicks for most. So one click for a small flag does not buy a small flag's refill,
// and a real change of flag slows or speeds the refill as the clicks for the new one outweigh the old.
type Toll struct {
	steps  []TollStep
	shares ShareReader
	flags  Flags
	clock  cptime.Clock
}

// PriceFor is the price of payer's next click for country: its main flag's once that click counts. The flag is the
// account's, or the scope's with no account, as its own bucket is.
func (t *Toll) PriceFor(payer Payer, country string) Price {
	allegiance := t.flags.OfScope(payer.Scope)
	if payer.Account != "" {
		allegiance = t.flags.OfAccount(payer.Account)
	}

	return t.Price(allegiance.With(country, t.clock.Now()).Flag())
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
