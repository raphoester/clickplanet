package ledger

import (
	"sort"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Replayed reads the world's ledger from a blank map: each event in position order, each reset and each cut.
type Replayed interface {
	Done(entry Entry) bool
	Record(entry Entry) error
	Wipe(at time.Time)
	Cut(at time.Time, board clicks.BoardID, carry Carry)
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

// Undoing is the world's map at the end of its ledger twice: with a caller's events, and as if it never made them. A
// revert gives back each tile where the two differ and the live map shows the first.
type Undoing struct {
	caller        Caller
	with, without *ending
}

func NewUndoing(caller Caller, first clicks.Stage) *Undoing {
	return &Undoing{caller: caller, with: newEnding(first), without: newEnding(first)}
}

var _ Replayed = (*Undoing)(nil)

func (u *Undoing) Done(Entry) bool { return false }

// Record reads the caller's events in both maps whatever an earlier revert forgot of them, so a revert run again gives
// back what the one before could not reach.
func (u *Undoing) Record(entry Entry) error {
	with, without := entry, entry
	if u.caller.Made(Taking{Scope: entry.Scope, Account: entry.Account}) {
		with.Forgotten, without.Forgotten = false, true
	}
	if err := u.with.record(with); err != nil {
		return err
	}
	return u.without.record(without)
}

func (u *Undoing) Wipe(time.Time) {
	u.with.wipe()
	u.without.wipe()
}

func (u *Undoing) Cut(_ time.Time, board clicks.BoardID, carry Carry) {
	u.with.cut(board, carry)
	u.without.cut(board, carry)
}

// Touched counts the tiles of the map in play the caller's events changed.
func (u *Undoing) Touched() int {
	touched := 0
	u.differences(func(uint32, string, string) { touched++ })
	return touched
}

// Restorations gives each tile the caller's events changed back to what it would show without them, if the live map
// still shows what they made of it, and only on the map in play.
func (u *Undoing) Restorations(owners Owners, playing clicks.Stage) []clicks.Restoration {
	if u.with.replayed != playing {
		return nil
	}
	var restorations []clicks.Restoration
	u.differences(func(tile uint32, with, without string) {
		if owner, _ := owners.Owner(tile); owner == with {
			restorations = append(restorations, clicks.Restoration{Tile: tile, From: with, To: without})
		}
	})
	sort.Slice(restorations, func(i, j int) bool { return restorations[i].Tile < restorations[j].Tile })
	return restorations
}

func (u *Undoing) differences(see func(tile uint32, with, without string)) {
	for tile, with := range u.with.owners {
		if without := u.without.owners[tile]; without != with {
			see(tile, with, without)
		}
	}
	for tile, without := range u.without.owners {
		if _, held := u.with.owners[tile]; !held {
			see(tile, "", without)
		}
	}
}

// ending is the world's map at the end of the ledger, read as the season's history reads it.
type ending struct {
	replayed clicks.Stage
	owners   map[uint32]string
	erasure  *Erasure
}

func newEnding(first clicks.Stage) *ending {
	return &ending{replayed: first, owners: map[uint32]string{}, erasure: NewErasure()}
}

func (e *ending) record(entry Entry) error {
	if entry.Stage != e.replayed {
		_, err := EventOf(entry)
		return err
	}
	event, kept, err := e.erasure.Read(entry)
	if err != nil || !kept {
		return err
	}
	event.Replay(func(taking Taking) {
		if taking.Country == "" {
			delete(e.owners, taking.Tile)
			return
		}
		e.owners[taking.Tile] = taking.Country
	})
	return nil
}

func (e *ending) wipe() {
	clear(e.owners)
	e.erasure.Wipe()
}

func (e *ending) cut(board clicks.BoardID, carry Carry) {
	before := e.owners
	e.owners = map[uint32]string{}
	carry.Each(func(child, parent uint32) {
		if country := before[parent]; country != "" {
			e.owners[child] = country
		}
	})
	e.replayed.Board = board
	e.erasure.Wipe()
}
