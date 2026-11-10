package clicks

import (
	"errors"
	"time"
)

var ErrNoKnockOut = errors.New("the take fells no standing flag's capital")

// KnockOut is a flag out: another flag took its capital, and every other tile it held went blank.
type KnockOut struct {
	Country string
	By      string
	// The capital, and the account whose take felled it.
	Tile    uint32
	Account string
	At      time.Time
	Cleared []uint32
}

// KnockOutError is nil when a take of a tile knocks a flag out: while the knock-out plays, the capital of a flag not out
// yet, taken by another flag. It reads the capital's flag, not the tile's last owner, so a capital left blank still falls.
func (r CapitalRules) KnockOutError(capitalOf string, out bool, flag string) error {
	if !r.KnockOut || capitalOf == "" || out || capitalOf == flag {
		return ErrNoKnockOut
	}
	return nil
}

// Stands is true for a flag still in the game: every flag while the knock-out does not play, and while it does, a flag
// with a capital that is not out.
func (r CapitalRules) Stands(capital, out bool) bool {
	return !r.KnockOut || (capital && !out)
}
