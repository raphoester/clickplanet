package clicks

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// ErrNotTrappable is a trap laid on a tile that is not its flag's, or holds a trap already: it stays in the hand.
var ErrNotTrappable = errors.New("a trap is laid only on a tile of its flag with no trap")

// Trap waits on a tile of its flag. A take by another flag springs it: the take's player loses clicks from its bank, to
// nobody, and the trap is gone. Any other change of owner takes it off the map with no effect.
type Trap struct {
	tile    uint32
	flag    string
	account cpsession.AccountID
}

type TrapSnapshot struct {
	Tile uint32
	// The flag that laid it: the tile's as it was laid, and as long as it is on the map.
	Flag string
	// Who laid it: a deleted account's traps leave the map.
	Account cpsession.AccountID
}

func (s TrapSnapshot) Restore() Trap {
	return Trap{tile: s.Tile, flag: s.Flag, account: s.Account}
}

func (t Trap) TakeSnapshot() TrapSnapshot {
	return TrapSnapshot{Tile: t.tile, Flag: t.flag, Account: t.account}
}

// NewTrap is a trap a player of the flag lays on the tile, which is not on the map yet.
func NewTrap(account cpsession.AccountID, flag string, tile uint32) Trap {
	return Trap{tile: tile, flag: flag, account: account}
}

// Laid is the trap on its tile, which must be its flag's and hold no trap. A mine may wait there too.
func (t Trap) Laid(tile Tile, trapped bool) (Trap, error) {
	if tile.id != t.tile || tile.owner != t.flag || trapped {
		return Trap{}, ErrNotTrappable
	}

	return t, nil
}

// Snare is what a trap sprung costs the take's player.
type Snare interface {
	// Spring has the payer pay for the trap its take sprang, and answers what left its own bank.
	Spring(payer Payer, trap TrapSnapshot) float64
}

// taker is who a take is for the traps it springs: the player who pays for them.
type taker struct {
	payer Payer
	snare Snare
}

// tripped is the take once it set off its tile's mine and sprang its trap, which its player paid for: a take by another
// flag is landing on the tile.
func (t taker) tripped(tiles Claimable, impact Impact) Impact {
	trap, ok := tiles.Trip(impact.Tile)
	if !ok {
		return impact
	}
	kept := trap.TakeSnapshot()
	t.snare.Spring(t.payer, kept)
	impact.Sprung = &kept

	return impact
}

// Sprung is every trap the takes sprang, in order.
func Sprung(impacts []Impact) []TrapSnapshot {
	var sprung []TrapSnapshot
	for _, impact := range impacts {
		if impact.Sprung != nil {
			sprung = append(sprung, *impact.Sprung)
		}
	}

	return sprung
}

type TrappableTiles interface {
	// Lay holds the lock through `write`, and answers the map's version under it: a lay changes no tile.
	Lay(ctx context.Context, write func(tiles Laying) error) (MapVersion, error)
}

func NewTrapline(tiles TrappableTiles) Trapline {
	return Trapline{tiles: tiles}
}

// Trapline lays the traps on the map, each under one lock of the tiles. A take springs one in the claim (Claimable.Trip).
type Trapline struct {
	tiles TrappableTiles
}

// Lay puts the trap on its tile, which must be its flag's and hold no trap, and answers the map's version.
func (l Trapline) Lay(ctx context.Context, trap Trap) (MapVersion, error) {
	return l.tiles.Lay(ctx, func(tiles Laying) error { //nolint:wrapcheck // a pure delegation: the storage names it.
		tile, err := tiles.Tile(trap.tile)
		if err != nil {
			return err //nolint:wrapcheck // a pure delegation: the storage already named what failed.
		}
		laid, err := trap.Laid(tile, tiles.Trapped(trap.tile))
		if err != nil {
			return err
		}
		tiles.PutTrap(laid)

		return nil
	})
}
