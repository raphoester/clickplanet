package paint

import (
	"cmp"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

// One tile a player took for a flag, as planet tells it.
type Take struct {
	Account cpsession.AccountID
	Tile    drawings.TileID
	Flag    string
	At      time.Time
}

// What one player painted with one flag in the window: each tile once, smallest first.
type Painting struct {
	Account cpsession.AccountID
	Flag    string
	Tiles   []drawings.TileID
}

type Clock interface {
	Now() time.Time
}

// The takes of the last span, each player's latest of each tile, in memory: a restart starts it again from what the
// stream still holds. A take may come twice or late: it is kept once, and one older than the span is not kept.
type Window struct {
	span  time.Duration
	clock Clock

	mu    sync.Mutex
	takes map[cpsession.AccountID]map[drawings.TileID]mark
}

type mark struct {
	flag string
	at   time.Time
}

func NewWindow(span time.Duration, clock Clock) *Window {
	return &Window{span: span, clock: clock, takes: map[cpsession.AccountID]map[drawings.TileID]mark{}}
}

func (w *Window) Keep(take Take) {
	if take.Account == cpsession.NoAccount || take.At.Before(w.clock.Now().Add(-w.span)) {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	tiles, ok := w.takes[take.Account]
	if !ok {
		tiles = map[drawings.TileID]mark{}
		w.takes[take.Account] = tiles
	}
	if kept, ok := tiles[take.Tile]; ok && kept.at.After(take.At) {
		return
	}
	tiles[take.Tile] = mark{flag: take.Flag, at: take.At}
}

// Every painting of the span of least tiles or more, by account then flag.
func (w *Window) Paintings(least int) []Painting {
	since := w.clock.Now().Add(-w.span)
	w.mu.Lock()
	defer w.mu.Unlock()

	var paintings []Painting
	for account, tiles := range w.takes {
		byFlag := map[string][]drawings.TileID{}
		for tile, kept := range tiles {
			if !kept.at.Before(since) {
				byFlag[kept.flag] = append(byFlag[kept.flag], tile)
			}
		}
		for flag, painted := range byFlag {
			if len(painted) >= least {
				slices.Sort(painted)
				paintings = append(paintings, Painting{Account: account, Flag: flag, Tiles: painted})
			}
		}
	}
	slices.SortFunc(paintings, func(a, b Painting) int {
		if c := cmp.Compare(a.Account.String(), b.Account.String()); c != 0 {
			return c
		}
		return cmp.Compare(a.Flag, b.Flag)
	})
	return paintings
}

// Drops every take older than the span, and every player left with none.
func (w *Window) Forget() {
	since := w.clock.Now().Add(-w.span)
	w.mu.Lock()
	defer w.mu.Unlock()

	for account, tiles := range w.takes {
		for tile, kept := range tiles {
			if kept.at.Before(since) {
				delete(tiles, tile)
			}
		}
		if len(tiles) == 0 {
			delete(w.takes, account)
		}
	}
}
