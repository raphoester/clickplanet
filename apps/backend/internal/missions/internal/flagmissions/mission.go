package flagmissions

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNoOrder = errors.New("a flag mission has no order")

type MissionID uuid.UUID

func (id MissionID) String() string { return uuid.UUID(id).String() }

// A tile taken for Flag on Ground: Position in the ledger, Index in its spread or enclose.
type Take struct {
	Flag     string
	Ground   string
	Position uint64
	Index    uint32
	At       time.Time
}

// A shield of Owner that a click for Flag broke.
type Strike struct {
	Flag  string
	Owner string
	At    time.Time
}

// What the whole flag is to do in a day: its own fields, and one progress from 0. A new kind is a new type.
type Order interface {
	advancedByTake(progress uint32, take Take) uint32
	advancedByStrike(progress uint32, strike Strike) uint32
	done(progress uint32) bool
}

// Take Goal tiles on Ground, a country's land.
type TakeIn struct {
	Ground string
	Goal   uint32
}

func (o TakeIn) advancedByTake(progress uint32, take Take) uint32 {
	return counted(progress, o.Goal, take.Ground == o.Ground)
}
func (o TakeIn) advancedByStrike(progress uint32, _ Strike) uint32 { return progress }
func (o TakeIn) done(progress uint32) bool                         { return progress >= o.Goal }

// Break Goal shields of Flag.
type BreakShields struct {
	Flag string
	Goal uint32
}

func (o BreakShields) advancedByTake(progress uint32, _ Take) uint32 { return progress }
func (o BreakShields) advancedByStrike(progress uint32, strike Strike) uint32 {
	return counted(progress, o.Goal, strike.Owner == o.Flag)
}
func (o BreakShields) done(progress uint32) bool { return progress >= o.Goal }

func counted(progress, goal uint32, counts bool) uint32 {
	if counts && progress < goal {
		return progress + 1
	}
	return progress
}

type MissionSnapshot struct {
	ID     MissionID
	Order  Order
	Points uint32
}
