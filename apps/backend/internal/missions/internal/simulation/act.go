package simulation

import (
	"time"
)

// What the ledger kept of a season that moves a flag's mission or the map, in ledger order: Position is the event's,
// Index the act's place among the acts of its kind in it, as a spread's takes or a bomb's strikes.
type Act interface {
	happened() time.Time
}

// A tile taken for Flag from Previous. Account is false for a take nobody signed, which planet never tells.
type Take struct {
	Position uint64
	Index    uint32
	At       time.Time
	Flag     string
	Previous string
	Tile     uint32
	Account  bool
}

// A shield of Owner that Flag broke: the tile keeps its flag.
type Strike struct {
	Position uint64
	Index    uint32
	At       time.Time
	Flag     string
	Owner    string
	Tile     uint32
}

// A tile a bomb left with no flag.
type Clear struct {
	Position uint64
	Index    uint32
	At       time.Time
	Previous string
	Tile     uint32
}

func (t Take) happened() time.Time   { return t.At }
func (s Strike) happened() time.Time { return s.At }
func (c Clear) happened() time.Time  { return c.At }
