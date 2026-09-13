// Package toll prices a click by how much of the map its country already holds.
//
// The price is in tokens and is taken at the moment of the click, from the
// country clicked for. A slower refill for big countries would have been read
// off whatever country the caller played last, so a player could bank tokens
// on a small country and spend them on a big one.
package toll

import (
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// Step is one row of the table: from Share of the map, a click costs Cost
// tokens. Cost may be a fraction: 1.5 is half as slow again.
type Step struct {
	Share float64
	Cost  float64
}

// Config holds the steps, lowest share first. Empty prices every click at one token.
type Config struct {
	Steps []Step
}

// Validate refuses a table that could lock a country out: a cost above the
// burst is one no bucket can ever pay.
func (c Config) Validate(burst int) error {
	previous := Step{Cost: 1}

	for i, step := range c.Steps {
		if math.IsNaN(step.Share) || step.Share <= previous.Share || step.Share > 1 {
			return fmt.Errorf("toll.steps[%d].share is %v: shares must rise, above 0 and up to 1", i, step.Share)
		}
		if math.IsNaN(step.Cost) || step.Cost <= previous.Cost {
			return fmt.Errorf("toll.steps[%d].cost is %v: costs must rise, above 1", i, step.Cost)
		}
		if step.Cost > float64(burst) {
			return fmt.Errorf("toll.steps[%d].cost is %v, above rateLimiter.burst %d: nobody could ever click", i, step.Cost, burst)
		}
		previous = step
	}

	return nil
}

// Price is what a click for one country costs now, and what it will cost next.
type Price struct {
	Cost  float64
	Share float64

	// The share at which the next step starts, and its cost. Zero at the top step.
	NextShare float64
	NextCost  float64
}

// Budget is an allowance counted in clicks at a price, rather than in tokens.
type Budget struct {
	cpratelimit.State
	Price Price
}

// Of reads a bucket in clicks: at a cost of 2, ten tokens refilling at 1/s are
// five clicks refilling at 0.5/s, so the meter on screen narrows with no client
// arithmetic, the way a bonus widens it. The capacity rounds down, so the meter
// never shows a click the bucket cannot hold.
func Of(state cpratelimit.State, price Price) Budget {
	cost := max(price.Cost, 1)

	return Budget{
		State: cpratelimit.State{
			Tokens:    state.Tokens / cost,
			Capacity:  int(float64(state.Capacity) / cost),
			PerSecond: state.PerSecond / cost,
		},
		Price: price,
	}
}

type ShareReader interface {
	Share(country string) float64
}

func New(config Config, shares ShareReader) *Toll {
	return &Toll{steps: config.Steps, shares: shares}
}

type Toll struct {
	steps  []Step
	shares ShareReader
}

func (t *Toll) Price(country string) Price {
	price := Price{Cost: 1, Share: t.shares.Share(country)}

	for _, step := range t.steps {
		if price.Share < step.Share {
			price.NextShare, price.NextCost = step.Share, step.Cost
			break
		}
		price.Cost = step.Cost
	}

	return price
}
