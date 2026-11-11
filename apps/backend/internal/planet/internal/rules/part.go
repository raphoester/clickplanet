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
	Key string
	// Path names the part's reasons by the message of planet.v1.Rules it reads, when that is not Key: two parts read
	// bonuses, the half that grants charges and the half that holds them.
	Path string
	Read func(*planetv1.Rules) R
	// InDefault, when set, refuses what default itself may not play: every other set would play it too.
	InDefault func(R) error
}

func (p PartOf[R]) key() string { return p.Key }

func (p PartOf[R]) path() string {
	if p.Path == "" {
		return p.Key
	}
	return p.Path
}

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
	path() string
	read(message *planetv1.Rules, isDefault bool) (any, error)
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
