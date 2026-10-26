package clicks

import (
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

type TollInForce interface {
	Toll() []TollStep
}

func NewToll(rules TollInForce, shares ShareReader) *Toll {
	return &Toll{rules: rules, shares: shares}
}

type Toll struct {
	rules  TollInForce
	shares ShareReader
}

func (t *Toll) Steps() []TollStep {
	return slices.Clone(t.rules.Toll())
}

func (t *Toll) Price(country string) Price {
	price := Price{Slowdown: 1, Share: t.shares.Share(country)}

	for _, step := range t.rules.Toll() {
		if price.Share < step.Share {
			price.NextShare, price.NextSlowdown = step.Share, step.Slowdown
			break
		}
		price.Slowdown = step.Slowdown
	}

	return price
}
