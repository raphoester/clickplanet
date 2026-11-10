package moments

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

// Action is what a command asks of scoring; at one instant they are sent in this order.
type Action uint8

const (
	OpenBook Action = iota + 1
	CloseRound
	OpenRound
	CloseBook
)

func (a Action) String() string {
	switch a {
	case OpenBook:
		return "open book"
	case CloseRound:
		return "close round"
	case OpenRound:
		return "open round"
	case CloseBook:
		return "close book"
	}
	return "unknown"
}

// Command is what the gamemaster asks scoring at a time: At is the start of what it opens, the end of what it closes.
type Command struct {
	Action   Action
	Book     BookID
	Round    RoundID
	Times    schedule.Times
	At       time.Time
	Tiebreak RoundID
	// FromNothing is an OpenRound of an arena battle: its round counts from nothing, on the blank map.
	FromNothing bool
	Ranking     Ranking
}

// Ranking is how scoring places an OpenRound's countries: a day on the ground they hold and the ground they gained, so
// holding ground pays; a battle, which shuffles the cards, on the ground gained alone.
type Ranking string

const (
	RankedByGain        Ranking = "gain"
	RankedByHeldAndGain Ranking = "held and gain"
)

// Key is a command's identity: an open is sent once per process, a close again until scoring answers it.
type Key struct {
	Action Action
	Book   BookID
	Round  RoundID
}

func (c Command) Key() Key { return Key{Action: c.Action, Book: c.Book, Round: c.Round} }

// Closes is true for the commands scoring answers with a fact: each is sent again until the fact comes.
func (a Action) Closes() bool { return a == CloseRound || a == CloseBook }
