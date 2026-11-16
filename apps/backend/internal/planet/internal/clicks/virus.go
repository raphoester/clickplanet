package clicks

import (
	"errors"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var (
	// ErrNotInfectable is an aim at a tile of no other flag, or with a shield: the virus stays in the hand.
	ErrNotInfectable = errors.New("a virus takes only an enemy tile with no shield")

	ErrNoVirus = errors.New("no such virus on the map")
)

type VirusID string

// Virus spreads into the land of the flag it hit. Each tile it infected infects one tile of that flag next to it at each
// tick, until it made its most infections, a take or a struck shield each. A change of owner cures a tile.
type Virus struct {
	id         VirusID
	account    cpsession.AccountID
	scope      string
	flag       string
	victim     string
	tile       uint32
	infected   []uint32
	infections int
	next       time.Time
	ended      bool
}

type VirusSnapshot struct {
	ID      VirusID
	Account cpsession.AccountID
	Scope   string
	// The flag it takes tiles for, and the flag whose land it spreads into.
	Flag   string
	Victim string
	// The tile it was used on.
	Tile uint32
	// The tiles it infected that no change of owner cured since, oldest first.
	Infected   []uint32
	Infections int
	Next       time.Time
	Ended      bool
}

func (s VirusSnapshot) Restore() Virus {
	return Virus{
		id: s.ID, account: s.Account, scope: s.Scope, flag: s.Flag, victim: s.Victim, tile: s.Tile,
		infected: slices.Clone(s.Infected), infections: s.Infections, next: s.Next, ended: s.Ended,
	}
}

func (v Virus) TakeSnapshot() VirusSnapshot {
	return VirusSnapshot{
		ID: v.id, Account: v.account, Scope: v.scope, Flag: v.flag, Victim: v.victim, Tile: v.tile,
		Infected: slices.Clone(v.infected), Infections: v.infections, Next: v.next, Ended: v.ended,
	}
}

// NewVirus is a virus aimed at a tile, which has hit no flag yet.
func NewVirus(id VirusID, account cpsession.AccountID, scope, flag string, tile uint32) Virus {
	return Virus{id: id, account: account, scope: scope, flag: flag, tile: tile}
}

// Aimed is the virus with the flag its tile wears as the one it hits.
func (v Virus) Aimed(tile Tile) (Virus, error) {
	if tile.id != v.tile || tile.owner == "" || tile.owner == v.flag || tile.shields > 0 {
		return Virus{}, ErrNotInfectable
	}
	v.victim = tile.owner

	return v, nil
}

func (v Virus) Due(now time.Time) bool {
	return !v.ended && !now.Before(v.next)
}

func (v Virus) spent(most int) bool {
	return v.infections >= most
}

// infecting is each tile that spreads at this tick: the ones it infected before it.
func (v Virus) infecting() []uint32 {
	return slices.Clone(v.infected)
}

// Infected counts one infection, and a tile it took infects from the next tick.
func (v Virus) Infected(impact Impact) Virus {
	v.infections++
	if impact.Outcome == Taken {
		v.infected = append(slices.Clone(v.infected), impact.Tile)
	}

	return v
}

// Ticked ends a virus that infected nothing at this tick, holds no tile any more, or made its most infections; one that
// goes on spreads next at next.
func (v Virus) Ticked(spread bool, most int, next time.Time) Virus {
	v.next = next
	v.ended = !spread || len(v.infected) == 0 || v.spent(most)

	return v
}

// Cured is the virus with the tile gone from it: the tile changed hands.
func (v Virus) Cured(tile uint32) Virus {
	v.infected = slices.DeleteFunc(slices.Clone(v.infected), func(infected uint32) bool { return infected == tile })

	return v
}

// Infection is one use or tick of a virus on the map: what it took and struck, and the virus after it.
type Infection struct {
	Virus  VirusSnapshot
	Taken  []uint32
	Struck []TileShields
}

type InfectedTile struct {
	Tile  uint32
	Virus VirusID
}
