package tempo

import (
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Switches struct {
	rules atomic.Pointer[Rules]
}

func NewSwitches() *Switches {
	s := &Switches{}
	s.Set(Plain())
	return s
}

func (s *Switches) Set(rules Rules) {
	s.rules.Store(&rules)
}

func (s *Switches) Rules() Rules {
	return *s.rules.Load()
}

type Pricer interface {
	Price(country string) clicks.Price
}

func NewPricing(toll Pricer, switches *Switches) Pricing {
	return Pricing{toll: toll, switches: switches}
}

type Pricing struct {
	toll     Pricer
	switches *Switches
}

func (p Pricing) Price(country string) clicks.Price {
	price := p.toll.Price(country)
	price.Speedup = p.switches.Rules().RefillMultiplier()
	return price
}
