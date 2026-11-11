package cprules

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

type Rules interface {
	// Validate names each value it refuses by its path in the part, or in the whole message for a Root part.
	Validate() error
}

// PartOf is one concept's rules in a set, read out of the set's message M by its own package.
type PartOf[M proto.Message, R Rules] struct {
	Key  string
	Read func(M) R
	// Root, when set, is a part whose Validate names each value by its whole path in the message: its reasons take no
	// Key before them.
	Root bool
	// InDefault, when set, refuses what default itself may not play: every other set would play it too.
	InDefault func(R) error
}

func (p PartOf[M, R]) key() string { return p.Key }

// read is the part as the message plays it, and every reason it is refused, each named by its path in the message.
func (p PartOf[M, R]) read(message M, isDefault bool) (any, []string) {
	rules := p.Read(message)
	err := rules.Validate()
	if isDefault && p.InDefault != nil {
		err = errors.Join(err, p.InDefault(rules))
	}
	if p.Root {
		return rules, reasonsOf("", err)
	}
	return rules, reasonsOf(p.Key+".", err)
}

type Part[M proto.Message] interface {
	key() string
	read(message M, isDefault bool) (any, []string)
}

type Parts[M proto.Message] []Part[M]

// Of is the part as the set plays it.
func (p PartOf[M, R]) Of(set Set[M]) R {
	return set.parts[p.Key].(R) //nolint:forcetypeassert // Restore keeps each part as its R.
}

// Reader reads the part of the set in force each time it is called: a set taken applies from the next call.
func Reader[M proto.Message, R Rules](inForce *InForce[M], part PartOf[M, R]) func() R {
	return func() R {
		return part.Of(inForce.set.Load().set)
	}
}
