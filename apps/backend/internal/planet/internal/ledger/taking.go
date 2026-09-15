// Package ledger remembers every take of a tile, oldest first, for as long as the retention keeps it.
//
// A scope holds a tile when the tile's latest take is the scope's and the tile still wears that paint.
//
// A revert gives a held tile back to what it held before the scope's current run on it. The run walks
// back over the scope's own takes while each took the tile from the paint of the one before:
//
//	A il→ps, A ps→fr           back to il
//	A il→ps, B ps→de, A de→ps  back to de: B broke the run
//	A il→ps, B ps→de           A holds nothing
//	A il→ps, bomb, A ""→ps     back to nobody: a change the ledger never saw breaks the run
//
// A run reaches no further back than the retention, and a forgotten take is as if it never happened.
package ledger

import (
	"errors"
	"time"
)

// ErrInvalidScope is an operator naming a caller by something that is neither an address nor a scope.
var ErrInvalidScope = errors.New("not an address or a scope")

type Taking struct {
	Tile     uint32
	Scope    string
	Country  string
	Previous string
	At       time.Time
}

// Position is a take's place in the ledger; it survives a restart.
type Position uint64
