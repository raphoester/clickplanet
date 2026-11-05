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
}

// Key is a command's identity: each is sent once per process.
type Key struct {
	Action Action
	Book   BookID
	Round  RoundID
}

func (c Command) Key() Key { return Key{Action: c.Action, Book: c.Book, Round: c.Round} }
