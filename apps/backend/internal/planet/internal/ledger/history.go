package ledger

import (
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var ErrInvalidHistory = errors.New("a history starts before it ends, and moves by a step above none")

// Frame is the tiles whose owner changed since the frame before; the first frame holds every tile held.
type Frame struct {
	At     time.Time
	Owners map[uint32]Owner
}

// Owner is whom a tile went to, empty for no one, and when: its last change before the frame.
type Owner struct {
	Country string
	At      time.Time
}

// History replays the ledger's entries, in position order from its first one, into the world's map every step of a
// window, and each tile's changes of owner and shield hits during it. An entry made on an arena's map changes none of
// it: the world's map came back as it was.
type History struct {
	since, until time.Time
	step         time.Duration
	world        arenas.Played

	owners  map[uint32]Owner
	shown   map[uint32]string
	changed *cpcolls.Set[uint32]
	next    time.Time
	frames  []Frame
	hits    map[uint32]uint32
}

func NewHistory(since, until time.Time, step time.Duration, world arenas.Played) (*History, error) {
	if !since.Before(until) || step <= 0 {
		return nil, ErrInvalidHistory
	}
	return &History{
		since: since, until: until, step: step, world: world, next: since,
		owners: map[uint32]Owner{}, shown: map[uint32]string{}, changed: cpcolls.NewSet[uint32](), hits: map[uint32]uint32{},
	}, nil
}

// Done is true once an entry at or after the window's end was seen: the ones after it change nothing.
func (h *History) Done(entry Entry) bool {
	return !entry.At.Before(h.until)
}

func (h *History) Record(entry Entry) error {
	event, err := EventOf(entry)
	if err != nil {
		return err
	}
	if !h.world.Holds(entry.At) {
		return nil
	}
	h.frameUpTo(entry.At)
	during := !entry.At.Before(h.since)
	event.Replay(func(taking Taking) {
		h.owners[taking.Tile] = Owner{Country: taking.Country, At: entry.At}
		h.changed.Add(taking.Tile)
		if during {
			h.hits[taking.Tile]++
		}
	})
	if during {
		event.Strikes(func(strike Strike) { h.hits[strike.Tile]++ })
	}
	return nil
}

// Wipe empties the map, as a reset does with no event in the ledger. A wipe at a step's time is in that frame.
func (h *History) Wipe(at time.Time) {
	if !at.Before(h.until) {
		return
	}
	h.frameUpTo(at.Add(-time.Nanosecond))
	for tile, owner := range h.owners {
		if owner.Country != "" {
			h.owners[tile] = Owner{At: at}
			h.changed.Add(tile)
		}
	}
}

func (h *History) Frames() []Frame {
	h.frameUpTo(h.until)
	return h.frames
}

func (h *History) Hits() map[uint32]uint32 {
	return h.hits
}

// frameUpTo keeps a frame at each step not after at: an entry at a step's time is in the next frame.
func (h *History) frameUpTo(at time.Time) {
	for !h.next.After(at) && !h.next.After(h.until) {
		h.frames = append(h.frames, Frame{At: h.next, Owners: h.changedOwners()})
		h.next = h.next.Add(h.step)
		if h.next.After(h.until) && h.frames[len(h.frames)-1].At.Before(h.until) {
			h.next = h.until
		}
	}
}

func (h *History) changedOwners() map[uint32]Owner {
	owners := map[uint32]Owner{}
	if len(h.frames) == 0 {
		for tile, owner := range h.owners {
			if owner.Country != "" {
				owners[tile] = owner
				h.shown[tile] = owner.Country
			}
		}
		h.changed.Clear()
		return owners
	}
	h.changed.ForEach(func(tile uint32) {
		if owner := h.owners[tile]; owner.Country != h.shown[tile] {
			owners[tile] = owner
			h.shown[tile] = owner.Country
		}
	})
	h.changed.Clear()
	return owners
}
