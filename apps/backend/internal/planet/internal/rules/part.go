package rules

import (
	"errors"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Rules interface {
	Validate() error
}

// PartOf is one concept's rules in a set, read by its own package.
type PartOf[R Rules] struct {
	Key  string
	Read func(*planetv1.Rules) R
	// InDefault, when set, refuses what default itself may not play: every other set would play it too.
	InDefault func(R) error
}

func (p PartOf[R]) key() string { return p.Key }

func (p PartOf[R]) read(message *planetv1.Rules, isDefault bool) (any, error) {
	rules := p.Read(message)
	err := rules.Validate()
	if isDefault && p.InDefault != nil {
		err = errors.Join(err, p.InDefault(rules))
	}
	return rules, err
}

type Part interface {
	key() string
	read(message *planetv1.Rules, isDefault bool) (any, error)
}

type Parts []Part

func Reader[R Rules](inForce *InForce, part PartOf[R]) func() R {
	return func() R {
		return inForce.set.Load().set.parts[part.Key].(R) //nolint:forcetypeassert // Restore keeps each part as its R.
	}
}
