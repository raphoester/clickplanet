// Package ledger remembers every take of a tile, oldest first, for as long as the retention keeps it.
//
// A caller is a scope or an account. It holds a tile when the tile's latest take is its own and the tile
// still wears that paint.
//
// A revert gives a held tile back to what it held before the caller's current run on it. The run walks
// back over the caller's own takes while each took the tile from the paint of the one before:
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
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

var (
	// ErrInvalidScope is an operator naming a caller by something that is neither an address nor a scope.
	ErrInvalidScope   = errors.New("not an address or a scope")
	ErrInvalidAccount = errors.New("not an account id")
	ErrNoCaller       = errors.New("name a scope or an account, and only one")
)

type Taking struct {
	Tile  uint32
	Scope string
	// Account is the account the click token named, empty for none.
	Account  string
	Country  string
	Previous string
	At       time.Time
}

// Position is a take's place in the ledger; it survives a restart.
type Position uint64

// Caller is who an operator names: a scope or an account, never both.
type Caller struct {
	Scope   string
	Account string
}

// ParseCaller reads a scope from any address, or an account id, and refuses both or neither.
func ParseCaller(scope, account string) (Caller, error) {
	switch {
	case (scope == "") == (account == ""):
		return Caller{}, ErrNoCaller
	case account != "":
		id, err := uuid.Parse(account)
		if err != nil {
			return Caller{}, fmt.Errorf("%w: %q", ErrInvalidAccount, account)
		}
		return Caller{Account: id.String()}, nil
	}

	parsed, ok := cpipscope.Parse(scope)
	if !ok {
		return Caller{}, fmt.Errorf("%w: %q", ErrInvalidScope, scope)
	}

	return Caller{Scope: parsed}, nil
}

// Made says whether the take is the caller's.
func (c Caller) Made(taking Taking) bool {
	if c.Account != "" {
		return taking.Account == c.Account
	}

	return taking.Scope == c.Scope
}
