package clicks

import (
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// ErrNotMineable is a mine laid on a tile that is not its flag's, or holds a mine already: it stays in the hand.
var ErrNotMineable = errors.New("a mine is laid only on a tile of its flag with no mine")

// Mine waits on a tile of its flag. A take by another flag sets it off: its blast takes the tiles around it for the
// flag, and every mine the blast reaches goes off too. Any other change of owner takes it off the map with no blast.
type Mine struct {
	tile    uint32
	flag    string
	account cpsession.AccountID
	scope   string
}

type MineSnapshot struct {
	Tile uint32
	// The flag it takes tiles for: the tile's as it was laid, and as long as it is on the map.
	Flag    string
	Account cpsession.AccountID
	Scope   string
}

func (s MineSnapshot) Restore() Mine {
	return Mine{tile: s.Tile, flag: s.Flag, account: s.Account, scope: s.Scope}
}

func (m Mine) TakeSnapshot() MineSnapshot {
	return MineSnapshot{Tile: m.tile, Flag: m.flag, Account: m.account, Scope: m.scope}
}

// NewMine is a mine a player of the flag lays on the tile, which is not on the map yet.
func NewMine(account cpsession.AccountID, scope, flag string, tile uint32) Mine {
	return Mine{tile: tile, flag: flag, account: account, scope: scope}
}

// Laid is the mine on its tile, which must be its flag's and hold no mine.
func (m Mine) Laid(tile Tile, mined bool) (Mine, error) {
	if tile.id != m.tile || tile.owner != m.flag || mined {
		return Mine{}, ErrNotMineable
	}

	return m, nil
}

// Explosion is the mines a take set off, every mine a blast reached going off after it, in order.
type Explosion struct {
	Blasts []MineBlast
}

// MineBlast is one mine gone off: the tiles it took for its flag, and the ones that lost a shield and kept their flag.
type MineBlast struct {
	Mine   MineSnapshot
	Taken  []uint32
	Struck []TileShields
}

// Detonation is one mine gone off, with what its blast changed, for its player's record.
type Detonation struct {
	Mine    MineSnapshot
	Impacts []Impact
}
