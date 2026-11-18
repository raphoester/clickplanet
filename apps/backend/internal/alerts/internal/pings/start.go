package pings

import (
	"errors"
	"time"
)

var ErrUnknownStart = errors.New("no such start")

// What a start is told once by: the battle's round, or the season's book.
type StartKey string

type StartKind int

const (
	SeasonStart StartKind = iota + 1
	BattleStart
	FinaleStart
	// The world's map moved onto a larger board, its land carried over: Times is how many times the ground it has.
	WorldGrew
)

type Start struct {
	Key      StartKey
	Kind     StartKind
	Season   uint32
	Times    uint32
	StartsAt time.Time
	EndsAt   time.Time
}

func (s Start) Stale(now time.Time) bool {
	return now.Sub(s.StartsAt) > Fresh
}
