package ledger

import (
	"errors"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var ErrInvalidHistory = errors.New("a history starts before it ends, and moves by a step above none")

// Frame is the tiles whose owner changed since the frame before, or every tile held when it is whole: the window's
// first, and the first after a cut.
type Frame struct {
	At     time.Time
	Owners map[uint32]Owner
	Whole  bool
}

// Owner is whom a tile went to, empty for no one, and when: its last change before the frame.
type Owner struct {
	Country string
	At      time.Time
}

// EraTile is a tile of one era of a window: the era's place among the eras the window covers (0 for the one in force at
// its start), and the tile, on that era's board.
type EraTile struct {
	Era  int
	Tile uint32
}

// InEra is hits of one board, all in one era of the window.
func InEra(era int, hits map[uint32]uint32) map[EraTile]uint32 {
	kept := make(map[EraTile]uint32, len(hits))
	for tile, count := range hits {
		kept[EraTile{Era: era, Tile: tile}] = count
	}
	return kept
}

// History replays the ledger's entries, in position order from its first one, into one map every step of a window, on
// the board of the era in force at each time, and each tile's changes of owner and shield hits during it. An entry
// names its stage: one made on another map, or on another board than its time's era, changes none of it. A cut (the
// start of an era) empties the map onto the era's board, and the frame after it holds every tile held then, as the first
// frame does.
type History struct {
	since, until time.Time
	step         time.Duration
	replayed     clicks.Stage

	owners  map[uint32]Owner
	shown   map[uint32]string
	changed *cpcolls.Set[uint32]
	whole   bool
	era     int
	next    time.Time
	frames  []Frame
	hits    map[EraTile]uint32
}

// NewHistory replays the map on the board of the era in force before its first entry: Cut moves it to the next era's.
func NewHistory(since, until time.Time, step time.Duration, replayed clicks.Stage) (*History, error) {
	if !since.Before(until) || step <= 0 {
		return nil, ErrInvalidHistory
	}
	return &History{
		since: since, until: until, step: step, replayed: replayed, next: since, whole: true,
		owners: map[uint32]Owner{}, shown: map[uint32]string{}, changed: cpcolls.NewSet[uint32](), hits: map[EraTile]uint32{},
	}, nil
}

// Carry is a switch of board's carry-over.
type Carry interface {
	// Each visits each tile of the board after the switch with the tile of the board before it comes from. A tile with
	// none is left out: it starts blank.
	Each(visit func(child, parent uint32))
}

// Cut starts an era of the map on board at at: the frames before it are of the board before, each tile of board takes
// the owner of the tile carry says it comes from, and the first frame after it holds every tile held. An era that
// started after since is the next of the window's eras; one at or before it is in force at the window's start.
func (h *History) Cut(at time.Time, board clicks.BoardID, carry Carry) {
	if !at.Before(h.until) {
		return
	}
	h.frameUpTo(at.Add(-time.Nanosecond))
	h.replayed.Board = board
	before := h.owners
	h.owners = map[uint32]Owner{}
	carry.Each(func(child, parent uint32) {
		if country := before[parent].Country; country != "" {
			h.owners[child] = Owner{Country: country, At: at}
		}
	})
	clear(h.shown)
	h.changed.Clear()
	h.whole = true
	if at.After(h.since) {
		h.era++
	}
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
	if entry.Stage != h.replayed {
		return nil
	}
	h.frameUpTo(entry.At)
	during := !entry.At.Before(h.since)
	event.Replay(func(taking Taking) {
		h.owners[taking.Tile] = Owner{Country: taking.Country, At: entry.At}
		h.changed.Add(taking.Tile)
		if during {
			h.hits[EraTile{Era: h.era, Tile: taking.Tile}]++
		}
	})
	if during {
		event.Strikes(func(strike Strike) { h.hits[EraTile{Era: h.era, Tile: strike.Tile}]++ })
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

func (h *History) Hits() map[EraTile]uint32 {
	return h.hits
}

// frameUpTo keeps a frame at each step not after at: an entry at a step's time is in the next frame.
func (h *History) frameUpTo(at time.Time) {
	for !h.next.After(at) && !h.next.After(h.until) {
		whole := h.whole
		h.frames = append(h.frames, Frame{At: h.next, Owners: h.changedOwners(), Whole: whole})
		h.next = h.next.Add(h.step)
		if h.next.After(h.until) && h.frames[len(h.frames)-1].At.Before(h.until) {
			h.next = h.until
		}
	}
}

func (h *History) changedOwners() map[uint32]Owner {
	owners := map[uint32]Owner{}
	if h.whole {
		for tile, owner := range h.owners {
			if owner.Country != "" {
				owners[tile] = owner
				h.shown[tile] = owner.Country
			}
		}
		h.changed.Clear()
		h.whole = false
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
