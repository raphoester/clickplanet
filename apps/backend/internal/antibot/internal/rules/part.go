package rules

import (
	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
)

type Rules interface {
	Validate() error
}

// PartOf is one component's rules in a set: a watchdog's, the jury's or the ban book's, read by its own package.
type PartOf[R Rules] struct {
	Key  string
	Read func(*antibotv1.Rules) R
}

func (p PartOf[R]) key() string { return p.Key }

func (p PartOf[R]) read(message *antibotv1.Rules) (any, error) {
	rules := p.Read(message)
	return rules, rules.Validate()
}

type Part interface {
	key() string
	read(message *antibotv1.Rules) (any, error)
}

type Parts []Part

func Reader[R Rules](inForce *InForce, part PartOf[R]) func() R {
	return func() R {
		return inForce.set.Load().parts[part.Key].(R)
	}
}
