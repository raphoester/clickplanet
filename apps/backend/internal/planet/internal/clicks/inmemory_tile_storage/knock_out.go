package inmemory_tile_storage

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

// OutFlag is a flag the arena in play knocked out: kept with the map's tiles, saved by the same flush.
type OutFlag struct {
	Flag string
	By   string
	At   time.Time
}

// The flags out of the map in play, and the ones no flush saved yet.
type outFlags struct {
	out     map[string]OutFlag
	unsaved []OutFlag
}

func newOutFlags() *outFlags {
	return &outFlags{out: map[string]OutFlag{}}
}

func (o *outFlags) has(flag string) bool {
	_, ok := o.out[flag]
	return ok
}

func (o *outFlags) add(out OutFlag) {
	o.out[out.Flag] = out
	o.unsaved = append(o.unsaved, out)
}

// A stored flag put back at boot: nothing to flush.
func (o *outFlags) restore(out OutFlag) {
	o.out[out.Flag] = out
}

func (o *outFlags) flags() []string {
	return slices.Sorted(maps.Keys(o.out))
}

func (o *outFlags) drainUnsaved() []OutFlag {
	unsaved := o.unsaved
	o.unsaved = nil
	return unsaved
}

func (o *outFlags) markUnsaved(out []OutFlag) {
	o.unsaved = append(o.unsaved, out...)
}

// knockOutLocked knocks out the flag whose capital the take of tile felled: every other tile it holds goes blank, with
// no shield and no holder, and it is kept out.
func (s *Storage) knockOutLocked(ctx context.Context, tile uint32, flag string) (clicks.KnockOut, error) {
	country, err := s.capitals.Felled(tile, flag, s.out.has)
	if err != nil {
		return clicks.KnockOut{}, err //nolint:wrapcheck // the sentinel, unwrapped: every take asks.
	}

	var cleared []uint32
	if code, ok := s.board.codeOf(country); ok {
		left := s.board.heldBy(country)
		for id := uint32(1); left > 0 && id <= s.board.last(); id++ {
			if s.board.tiles[id].owner == code {
				s.moveLocked(id, unownedCode)
				cleared = append(cleared, id)
				left--
			}
		}
	}

	at := s.clock.Now().UTC()
	s.out.add(OutFlag{Flag: country, By: flag, At: at})

	return clicks.KnockOut{
		Country: country, By: flag, Tile: tile, Account: cpctx.GetAccount(ctx), At: at, Cleared: cleared,
	}, nil
}

// Stands says whether the flag is still in the game: always while the knock-out does not play, and while it does,
// while it has a capital that is not out.
func (s *Storage) Stands(flag string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.capitals.Stands(flag, s.out.has(flag))
}
