package retaker

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/evidence"
)

var _ evidence.Section = (*Watchdog)(nil)

type savedState struct {
	Tiles   []savedTake
	Callers []savedCaller
}

type savedTake struct {
	Tile  uint32
	Scope string
	At    int64
}

type savedCaller struct {
	Scope     string
	Reactions []savedReaction
	Tiles     []uint32
}

type savedReaction struct {
	At    int64
	Delay int64
	Tile  uint32
}

func (w *Watchdog) Save() ([]byte, error) {
	w.mu.Lock()

	saved := savedState{
		Tiles:   make([]savedTake, 0, len(w.tiles)),
		Callers: make([]savedCaller, 0, len(w.callers)),
	}
	for tile, t := range w.tiles {
		saved.Tiles = append(saved.Tiles, savedTake{Tile: tile, Scope: t.scope, At: evidence.Nanos(t.at)})
	}
	for scope, c := range w.callers {
		reactions := make([]savedReaction, 0, len(c.reactions))
		for _, r := range c.reactions {
			reactions = append(reactions, savedReaction{At: evidence.Nanos(r.at), Delay: int64(r.delay), Tile: r.tile})
		}
		saved.Callers = append(saved.Callers, savedCaller{
			Scope:     scope,
			Reactions: reactions,
			Tiles:     append([]uint32(nil), c.tiles...),
		})
	}

	w.mu.Unlock()

	return evidence.Encode(saved)
}

func (w *Watchdog) Load(data []byte) error {
	var saved savedState
	if err := evidence.Decode(data, &saved); err != nil {
		return err
	}

	tiles := make(map[uint32]take, len(saved.Tiles))
	for _, t := range saved.Tiles {
		tiles[t.Tile] = take{scope: t.Scope, at: evidence.Time(t.At)}
	}

	callers := make(map[string]*caller, len(saved.Callers))
	for _, c := range saved.Callers {
		loaded := &caller{tiles: c.Tiles}
		for _, r := range c.Reactions {
			loaded.reactions = append(loaded.reactions, reaction{at: evidence.Time(r.At), delay: time.Duration(r.Delay), tile: r.Tile})
		}
		if len(loaded.tiles) > keptTiles {
			loaded.tiles = loaded.tiles[len(loaded.tiles)-keptTiles:]
		}
		callers[c.Scope] = loaded
	}

	w.mu.Lock()
	w.tiles, w.callers = tiles, callers
	w.mu.Unlock()

	return nil
}

func (w *Watchdog) Forget(before time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for tile, t := range w.tiles {
		if t.at.Before(before) {
			delete(w.tiles, tile)
		}
	}

	for scope, c := range w.callers {
		c.prune(before)
		if len(c.reactions) == 0 {
			delete(w.callers, scope)
		}
	}
}
