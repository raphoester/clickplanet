package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
)

type SetName string

// Default is the set the migration keeps production's rules in, and the one played when no set has the name asked.
const Default SetName = "default"

var (
	ErrUnknownSet = errors.New("no rule set has this name")
	ErrInvalidSet = errors.New("this rule set cannot play")
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

type Set struct {
	name    SetName
	message *antibotv1.Rules
	parts   map[string]any
}

func (s Set) TakeSnapshot() Snapshot {
	return Snapshot{Name: s.name, Rules: proto.CloneOf(s.message)}
}

func (s Set) Is(name SetName) bool { return s.name == name }

// Equal is true when both sets have one name and judge alike.
func (s Set) Equal(other Set) bool {
	return s.name == other.name && proto.Equal(s.message, other.message)
}

type Snapshot struct {
	Name  SetName
	Rules *antibotv1.Rules
}

// Restore refuses a set any part refuses: a value left out reads as zero, and nothing falls back to a default.
func (s Snapshot) Restore(parts Parts) (Set, error) {
	if !namePattern.MatchString(string(s.Name)) {
		return Set{}, fmt.Errorf("%w: a name is a lowercase letter, then up to 31 of a-z, 0-9, - and _", ErrInvalidSet)
	}

	message := proto.CloneOf(s.Rules)
	read := make(map[string]any, len(parts))
	var reasons []string
	for _, part := range parts {
		rules, err := part.read(message)
		reasons = append(reasons, reasonsOf(part.key(), err)...)
		read[part.key()] = rules
	}
	if len(reasons) > 0 {
		return Set{}, fmt.Errorf("%w: %s", ErrInvalidSet, strings.Join(reasons, "; "))
	}

	return Set{name: s.Name, message: message, parts: read}, nil
}

// Each reason names its value by its path in the set, as antibot.v1.Rules writes it.
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
