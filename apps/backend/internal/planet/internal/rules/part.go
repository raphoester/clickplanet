package rules

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Rules interface {
	Validate() error
}

// PartOf is one concept's rules in a set, read by its own package.
type PartOf[R Rules] struct {
	Key  string
	Read func(*planetv1.Rules) R
}

func (p PartOf[R]) key() string { return p.Key }

func (p PartOf[R]) read(message *planetv1.Rules) (any, error) {
	rules := p.Read(message)
	return rules, rules.Validate()
}

type Part interface {
	key() string
	read(message *planetv1.Rules) (any, error)
}

type Parts []Part

func Reader[R Rules](inForce *InForce, part PartOf[R]) func() R {
	return func() R {
		return inForce.set.Load().set.parts[part.Key].(R) //nolint:forcetypeassert // Restore keeps each part as its R.
	}
}
