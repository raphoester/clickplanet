package clicks

import (
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
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

func NewToll(config TollConfig, shares ShareReader) *Toll {
	return &Toll{steps: config.Steps, shares: shares}
}

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
