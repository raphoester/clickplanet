package rules

import (
	bonusesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/bonuses/v1"
)

type Rules interface {
	// Validate names each value it refuses by its path in bonuses.v1.Rules, as protojson writes it.
	Validate() error
}

// PartOf is one concept's rules in a set, read by its own package.
type PartOf[R Rules] struct {
	Key  string
	Read func(*bonusesv1.Rules) R
}

func (p PartOf[R]) key() string { return p.Key }

func (p PartOf[R]) read(message *bonusesv1.Rules) (any, error) {
	rules := p.Read(message)
	return rules, rules.Validate()
}

type Part interface {
	key() string
	read(message *bonusesv1.Rules) (any, error)
}

type Parts []Part

// Of is the part as the set plays it.
func (p PartOf[R]) Of(set Set) R {
	return set.parts[p.Key].(R) //nolint:forcetypeassert // Restore keeps each part as its R.
}

func Reader[R Rules](inForce *InForce, part PartOf[R]) func() R {
	return func() R {
		return part.Of(inForce.set.Load().set)
	}
}
