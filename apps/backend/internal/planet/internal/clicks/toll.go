package clicks

import (
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
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
	Slowdown float64
	Share    float64

	// The share at which the next step starts, and its slowdown. Zero at the top step.
	NextShare    float64
	NextSlowdown float64
}

// Budget is an allowance: every click costs one token, so it is counted in clicks. Price is the country asked
// about's; LinkedMultiplier is what signing in multiplies the refill by, the same for every caller.
type Budget struct {
	cpratelimit.State
	Price            Price
	LinkedMultiplier float64
}

type ShareReader interface {
	Share(country string) float64
}

func NewToll(config TollConfig, shares ShareReader) *Toll {
	return &Toll{steps: config.Steps, shares: shares}
}

// A toll slows the refill of a player by how much of the map its country already holds.
//
// Every click costs one token; the price is the pace the bucket refills at afterwards, set by each click from the
// country it was for. The time already past was refilled at the pace in force over it, so switching flags moves
// nothing on the meter until the next click. A player can still refill on a small country and spend the bank on a
// big one; the bank bounds what that buys.
type Toll struct {
	steps  []TollStep
	shares ShareReader
}

func (t *Toll) Price(country string) Price {
	price := Price{Slowdown: 1, Share: t.shares.Share(country)}

	for _, step := range t.steps {
		if price.Share < step.Share {
			price.NextShare, price.NextSlowdown = step.Share, step.Slowdown
			break
		}
		price.Slowdown = step.Slowdown
	}

	return price
}
