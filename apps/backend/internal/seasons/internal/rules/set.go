package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
)

type SetName string

// Default is the set the migration keeps: no round of its own.
const Default SetName = "default"

// MaxTimes bounds a round of its own's points, so a slip of the keyboard cannot decide a season.
const MaxTimes = 10

var (
	ErrUnknownSet = errors.New("no rule set has this name")
	ErrInvalidSet = errors.New("this rule set cannot play")
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

type Set struct {
	name     SetName
	message  *seasonsv1.Rules
	ownRound OwnRound
}

// OwnRound is what a set asks of the window of the schedule entry that plays it.
type OwnRound struct {
	Enabled bool
	// Times is how many times a day's points the round scores.
	Times uint32
}

func (s Set) TakeSnapshot() Snapshot {
	return Snapshot{Name: s.name, Rules: proto.CloneOf(s.message)}
}

func (s Set) Is(name SetName) bool { return s.name == name }

// Equal is true when both sets have one name and count alike.
func (s Set) Equal(other Set) bool {
	return s.name == other.name && proto.Equal(s.message, other.message)
}

func (s Set) OwnRound() OwnRound { return s.ownRound }

type Snapshot struct {
	Name  SetName
	Rules *seasonsv1.Rules
}

// Restore refuses a set with a value it cannot count with, naming each by its path as seasons.v1.Rules writes it. A
// value left out reads as zero, and nothing falls back to a default.
func (s Snapshot) Restore() (Set, error) {
	if !namePattern.MatchString(string(s.Name)) {
		return Set{}, fmt.Errorf("%w: a name is a lowercase letter, then up to 31 of a-z, 0-9, - and _", ErrInvalidSet)
	}

	message := proto.CloneOf(s.Rules)
	own := OwnRound{Enabled: message.GetOwnRound().GetEnabled(), Times: message.GetOwnRound().GetTimes()}
	var reasons []string
	if own.Times < 1 || own.Times > MaxTimes {
		reasons = append(reasons, fmt.Sprintf("ownRound.times is %d, and must be 1 to %d", own.Times, MaxTimes))
	}
	if len(reasons) > 0 {
		return Set{}, fmt.Errorf("%w: %s", ErrInvalidSet, strings.Join(reasons, "; "))
	}

	return Set{name: s.Name, message: message, ownRound: own}, nil
}
