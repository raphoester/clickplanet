package clicks

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type TollStep struct {
	Share    float64
	Slowdown float64
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

func (b Budget) Empty() bool {
	return b.Tokens < 1
}

type ShareReader interface {
	Share(country string) float64
}

// TollSteps rise: from Share of every tile on the map, a country refills Slowdown times slower.
type TollSteps []TollStep

func (s TollSteps) Validate() error {
	previous := TollStep{Slowdown: 1}

	var problems []error
	for i, step := range s {
		if math.IsNaN(step.Share) || step.Share <= previous.Share || step.Share > 1 {
			problems = append(problems, fmt.Errorf("steps[%d].share is %v: shares rise, above 0 and up to 1", i, step.Share))
		}
		if math.IsNaN(step.Slowdown) || math.IsInf(step.Slowdown, 0) || step.Slowdown <= previous.Slowdown {
			problems = append(problems, fmt.Errorf("steps[%d].slowdown is %v: slowdowns rise, above 1", i, step.Slowdown))
		}
		previous = step
	}

	return errors.Join(problems...)
}

func NewToll(steps func() TollSteps, shares ShareReader) *Toll {
	return &Toll{steps: steps, shares: shares}
}

type Toll struct {
	steps  func() TollSteps
	shares ShareReader
}

func (t *Toll) Steps() []TollStep {
	return slices.Clone(t.steps())
}

func (t *Toll) Price(country string) Price {
	price := Price{Slowdown: 1, Share: t.shares.Share(country)}

	for _, step := range t.steps() {
		if price.Share < step.Share {
			price.NextShare, price.NextSlowdown = step.Share, step.Slowdown
			break
		}
		price.Slowdown = step.Slowdown
	}

	return price
}
