// Package cprules keeps a module's rules in named sets over a default, generic over the proto message the set holds:
// planet's, the antibot's and the offers' rule sets.
package cprules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpproto"
)

type SetName string

// Default is the set that holds every value: each other set plays default's for every value it leaves unset.
const Default SetName = "default"

var (
	ErrUnknownSet = errors.New("no rule set has this name")
	ErrInvalidSet = errors.New("this rule set cannot play")
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// Set is a set restored over default: the values it holds, the values it plays, and each part as it plays it.
type Set[M proto.Message] struct {
	name   SetName
	own    M
	played M
	parts  map[string]any
}

func (s Set[M]) TakeSnapshot() Snapshot[M] {
	return Snapshot[M]{Name: s.name, Rules: proto.CloneOf(s.own)}
}

// TakePlayedSnapshot is the set's values over default's.
func (s Set[M]) TakePlayedSnapshot() Snapshot[M] {
	return Snapshot[M]{Name: s.name, Rules: proto.CloneOf(s.played)}
}

func (s Set[M]) Is(name SetName) bool { return s.name == name }

// Equal is true when both sets have one name and play alike.
func (s Set[M]) Equal(other Set[M]) bool {
	return s.name == other.name && proto.Equal(s.played, other.played)
}

// Snapshot is a set as the wire and the store carry it: its name and the values it holds.
type Snapshot[M proto.Message] struct {
	Name  SetName
	Rules M
}

// Restore plays defaults' value wherever the set holds none; default plays alone, as it holds every value.
func (s Snapshot[M]) Restore(parts Parts[M], defaults Snapshot[M]) (Set[M], error) {
	if !namePattern.MatchString(string(s.Name)) {
		return Set[M]{}, fmt.Errorf("%w: a name is a lowercase letter, then up to 31 of a-z, 0-9, - and _", ErrInvalidSet)
	}

	own := orEmpty(s.Rules)
	played := proto.CloneOf(own)
	if s.Name != Default {
		played = cpproto.Overlay(orEmpty(defaults.Rules), played)
	}
	read := make(map[string]any, len(parts))
	var reasons []string
	for _, part := range parts {
		rules, refused := part.read(played, s.Name == Default)
		reasons = append(reasons, refused...)
		read[part.key()] = rules
	}
	if len(reasons) > 0 {
		return Set[M]{}, fmt.Errorf("%w: %s", ErrInvalidSet, strings.Join(reasons, "; "))
	}

	return Set[M]{name: s.Name, own: proto.CloneOf(own), played: played, parts: read}, nil
}

func orEmpty[M proto.Message](rules M) M {
	if rules.ProtoReflect().IsValid() {
		return rules
	}
	return empty[M]()
}

// empty is a new message of M's type, which holds no value.
func empty[M proto.Message]() M {
	var none M
	return none.ProtoReflect().Type().New().Interface().(M) //nolint:forcetypeassert // a new M is an M.
}

// Each reason names its value by its path in the set, as the message writes it: prefix is the part's key and a dot, or
// nothing for a part that names the whole path itself.
func reasonsOf(prefix string, err error) []string {
	if err == nil {
		return nil
	}

	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		return []string{prefix + err.Error()}
	}

	var reasons []string
	for _, each := range joined.Unwrap() {
		reasons = append(reasons, reasonsOf(prefix, each)...)
	}
	return reasons
}
