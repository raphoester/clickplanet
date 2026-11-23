package ledger

import (
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Erasure reads the ledger as if the events a revert forgot were never made.
type Erasure struct {
	painted map[stagedTile]painting
}

type stagedTile struct {
	stage clicks.Stage
	tile  uint32
}

type painting struct {
	under string
	paint string
}

func NewErasure() *Erasure {
	return &Erasure{painted: map[stagedTile]painting{}}
}

// Read takes each entry in position order: false for one forgotten, which shows nothing.
func (e *Erasure) Read(entry Entry) (Event, bool, error) {
	event, err := EventOf(entry)
	if err != nil {
		return nil, false, err
	}
	if entry.Forgotten {
		e.forget(entry.Stage, event)
		return nil, false, nil
	}
	return e.through(entry.Stage, event), true, nil
}

// Wipe is a change of every tile the ledger never saw: a reset blanks the map, a cut carries it onto another board.
func (e *Erasure) Wipe() {
	clear(e.painted)
}

func (e *Erasure) forget(on clicks.Stage, event Event) {
	event.Replay(func(taking Taking) {
		key := stagedTile{stage: on, tile: taking.Tile}
		painted, seen := e.painted[key]
		if !seen || taking.Previous != painted.paint {
			painted.under = taking.Previous
		}
		painted.paint = taking.Country
		if painted.under == painted.paint {
			delete(e.painted, key)
			return
		}
		e.painted[key] = painted
	})
}

func (e *Erasure) through(on clicks.Stage, event Event) Event {
	if len(e.painted) == 0 {
		return event
	}
	painted := map[uint32]painting{}
	look := func(tile uint32) {
		if was, ok := e.painted[stagedTile{stage: on, tile: tile}]; ok {
			painted[tile] = was
		}
	}
	event.Strikes(func(strike Strike) { look(strike.Tile) })
	event.Show(func(scene Scene) {
		if scene.Change != nil {
			look(scene.Change.Tile)
		}
	})
	event.Replay(func(taking Taking) {
		look(taking.Tile)
		delete(e.painted, stagedTile{stage: on, tile: taking.Tile})
	})
	if len(painted) == 0 {
		return event
	}
	return seenThrough{event: event, painted: painted}
}

type seenThrough struct {
	event   Event
	painted map[uint32]painting
}

// A tile that no longer wears the forgotten paint changed in a way the ledger never saw: a revert or a reset.
func (s seenThrough) under(tile uint32, wears string) (string, bool) {
	painted, ok := s.painted[tile]
	if !ok || painted.paint != wears {
		return "", false
	}
	return painted.under, true
}

func (s seenThrough) Replay(see func(Taking)) {
	s.event.Replay(func(taking Taking) {
		if under, painted := s.under(taking.Tile, taking.Previous); painted {
			if under == taking.Country {
				return
			}
			taking.Previous = under
		}
		see(taking)
	})
}

func (s seenThrough) Strikes(see func(Strike)) {
	s.event.Strikes(func(strike Strike) {
		if _, painted := s.under(strike.Tile, strike.Owner); !painted {
			see(strike)
		}
	})
}

func (s seenThrough) Show(see func(Scene)) {
	s.event.Show(func(scene Scene) {
		switch {
		case scene.Change != nil:
			change, shown := s.change(*scene.Change)
			if !shown {
				return
			}
			scene.Change = &change
		case scene.Blast != nil:
			blast := s.blast(*scene.Blast)
			scene.Blast = &blast
		case scene.Enclosure != nil:
			enclosure := *scene.Enclosure
			enclosure.Impacts = s.impacts(enclosure.Country, enclosure.Impacts)
			scene.Enclosure = &enclosure
		}
		see(scene)
	})
}

func (s seenThrough) Entry() (Entry, error) {
	entry, err := s.event.Entry()
	if err != nil {
		return Entry{}, fmt.Errorf("failed to write down an event read through what was forgotten: %w", err)
	}
	return entry, nil
}

func (s seenThrough) change(change Change) (Change, bool) {
	under, painted := s.under(change.Tile, change.Previous)
	switch {
	case !painted:
		return change, true
	case change.Value == change.Previous, change.Value == under:
		return change, false
	}
	change.Previous = under
	return change, true
}

func (s seenThrough) blast(blast clicks.Blast) clicks.Blast {
	cleared, owners := make([]uint32, 0, len(blast.Cleared)), make([]string, 0, len(blast.Owners))
	for i, tile := range blast.Cleared {
		owner := blast.Owners[i]
		if under, painted := s.under(tile, owner); painted {
			if under == "" {
				continue
			}
			owner = under
		}
		cleared, owners = append(cleared, tile), append(owners, owner)
	}
	struck, left, struckOwners := []uint32{}, []int{}, []string{}
	for i, tile := range blast.Struck {
		if _, painted := s.under(tile, blast.StruckOwners[i]); !painted {
			struck, left, struckOwners = append(struck, tile), append(left, blast.Left[i]), append(struckOwners, blast.StruckOwners[i])
		}
	}
	blast.Cleared, blast.Owners, blast.Struck, blast.Left, blast.StruckOwners = cleared, owners, struck, left, struckOwners
	return blast
}

func (s seenThrough) impacts(flag string, impacts []clicks.Impact) []clicks.Impact {
	kept := make([]clicks.Impact, 0, len(impacts))
	for _, impact := range impacts {
		under, painted := s.under(impact.Tile, impact.Owner)
		switch {
		case !painted:
		case impact.Outcome != clicks.Taken, under == flag:
			continue
		default:
			impact.Owner = under
		}
		kept = append(kept, impact)
	}
	return kept
}
