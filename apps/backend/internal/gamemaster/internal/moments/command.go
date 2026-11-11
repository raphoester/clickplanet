package moments

import (
	"errors"
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

// Key is a command's identity: a call is made until scoring answers it, once per process.
type Key struct {
	Action Action
	Book   BookID
	Round  RoundID
}

func (c Command) Key() Key { return Key{Action: c.Action, Book: c.Book, Round: c.Round} }

// ErrRefused is a call its receiver refused for good: it is not made again in this process.
var ErrRefused = errors.New("refused")

// Answer is what scoring answered a close: the end it scored the round or ranked the book at, or Missed: it never
// opened, so nothing scores it and nothing waits on it.
type Answer struct {
	Missed  bool
	EndedAt time.Time
}
