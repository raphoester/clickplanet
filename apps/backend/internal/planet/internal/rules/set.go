package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
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

type Set struct {
	name   SetName
	own    *planetv1.Rules
	played *planetv1.Rules
	parts  map[string]any
}

func (s Set) TakeSnapshot() Snapshot {
	return Snapshot{Name: s.name, Rules: proto.CloneOf(s.own)}
}

// TakePlayedSnapshot is the set's values over default's.
func (s Set) TakePlayedSnapshot() Snapshot {
	return Snapshot{Name: s.name, Rules: proto.CloneOf(s.played)}
}

func (s Set) Is(name SetName) bool { return s.name == name }

// Equal is true when both sets have one name and play alike.
func (s Set) Equal(other Set) bool {
	return s.name == other.name && proto.Equal(s.played, other.played)
}

type Snapshot struct {
	Name  SetName
	Rules *planetv1.Rules
}

// Restore plays defaults' value wherever the set holds none; default plays alone, as it holds every value.
func (s Snapshot) Restore(parts Parts, defaults Snapshot) (Set, error) {
	if !namePattern.MatchString(string(s.Name)) {
		return Set{}, fmt.Errorf("%w: a name is a lowercase letter, then up to 31 of a-z, 0-9, - and _", ErrInvalidSet)
	}

	played := proto.CloneOf(orEmpty(s.Rules))
	if s.Name != Default {
		played = cpproto.Overlay(orEmpty(defaults.Rules), played)
	}
	read := make(map[string]any, len(parts))
	var reasons []string
	for _, part := range parts {
		rules, err := part.read(played)
		reasons = append(reasons, reasonsOf(part.key(), err)...)
		read[part.key()] = rules
	}
	if len(reasons) > 0 {
		return Set{}, fmt.Errorf("%w: %s", ErrInvalidSet, strings.Join(reasons, "; "))
	}

	return Set{name: s.Name, own: proto.CloneOf(orEmpty(s.Rules)), played: played, parts: read}, nil
}

func orEmpty(rules *planetv1.Rules) *planetv1.Rules {
	if rules == nil {
		return &planetv1.Rules{}
	}
	return rules
}

// Each reason names its value by its path in the set, as planet.v1.Rules writes it.
func reasonsOf(part string, err error) []string {
	if err == nil {
		return nil
	}

	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		return []string{part + "." + err.Error()}
	}

	var reasons []string
	for _, each := range joined.Unwrap() {
		reasons = append(reasons, reasonsOf(part, each)...)
	}
	return reasons
}
