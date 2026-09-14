// Package ledger remembers, per tile, the last caller who took it and what the tile held before.
//
// It is what the operator tools read to say who is painting what, and to undo one caller's paint.
package ledger

import (
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// ErrInvalidScope is an operator naming a caller by something that is neither an address nor a scope.
var ErrInvalidScope = errors.New("not an address or a scope")

// Taking is one tile, as its last taker left it.
type Taking struct {
	Tile  uint32
	Scope string
	// Country is what the scope painted; Previous is what the tile held before the scope first took it.
	Country  string
	Previous string
	At       time.Time
}

// Over is this take recorded on top of the tile's last one. A scope that retakes its own tile
// keeps the owner from before its first take, so a revert goes back past all of them.
func (t Taking) Over(last Taking, found bool) Taking {
	if found && last.Scope == t.Scope {
		t.Previous = last.Previous
	}

	return t
}

// WornBy says whether a tile held by owner still wears this take's paint.
func (t Taking) WornBy(owner string) bool {
	return owner == t.Country
}

// Restoration gives the tile back to what it held before, only if it still wears this take's paint.
func (t Taking) Restoration() clicks.Restoration {
	return clicks.Restoration{Tile: t.Tile, From: t.Country, To: t.Previous}
}
