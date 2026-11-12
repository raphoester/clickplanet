package ledger

import (
	"encoding/binary"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Rewind is a History's frames rewound from the map now, so it reads the entries from since on and none before.
type Rewind struct {
	at    []time.Time
	world arenas.Played

	rewound map[uint32]string
	steps   []map[uint32]Owner
	hits    map[uint32]uint32
}

func NewRewind(since, until time.Time, step time.Duration, world arenas.Played) (*Rewind, error) {
	if until.Before(since) || step <= 0 {
		return nil, ErrInvalidHistory
	}
	at := []time.Time{since}
	for next := since.Add(step); next.Before(until); next = next.Add(step) {
		at = append(at, next)
	}
	if since.Before(until) {
		at = append(at, until)
	}
	steps := make([]map[uint32]Owner, len(at))
	for i := range steps {
		steps[i] = map[uint32]Owner{}
	}
	return &Rewind{at: at, world: world, rewound: map[uint32]string{}, steps: steps, hits: map[uint32]uint32{}}, nil
}

// Record takes the entries in position order: a tile opens on what it held before its first change at or after since.
func (r *Rewind) Record(entry Entry) error {
	event, err := EventOf(entry)
	if err != nil {
		return err
	}
	if entry.At.Before(r.at[0]) || !r.world.Holds(entry.At) {
		return nil
	}
	frame, during := r.frameOf(entry.At)
	event.Replay(func(taking Taking) {
		if _, seen := r.rewound[taking.Tile]; !seen {
			r.rewound[taking.Tile] = taking.Previous
		}
		if during {
			r.steps[frame][taking.Tile] = Owner{Country: taking.Country, Account: taking.Account, At: entry.At}
			r.hits[taking.Tile]++
		}
	})
	if during {
		event.Strikes(func(strike Strike) { r.hits[strike.Tile]++ })
	}
	return nil
}

// An entry at a step's time is in the next frame, as in a History.
func (r *Rewind) frameOf(at time.Time) (int, bool) {
	for frame := 1; frame < len(r.at); frame++ {
		if at.Before(r.at[frame]) {
			return frame, true
		}
	}
	return 0, false
}

func (r *Rewind) Frames(now clicks.DenseBatch) []Frame {
	opening := r.openingOf(now)
	shown := make(map[uint32]string)
	frames := []Frame{{At: r.at[0], Owners: opening}}
	for frame := 1; frame < len(r.at); frame++ {
		owners := map[uint32]Owner{}
		for tile, owner := range r.steps[frame] {
			was, seen := shown[tile]
			if !seen {
				was = opening[tile].Country
			}
			if owner.Country != was {
				owners[tile] = owner
				shown[tile] = owner.Country
			}
		}
		frames = append(frames, Frame{At: r.at[frame], Owners: owners})
	}
	return frames
}

func (r *Rewind) openingOf(now clicks.DenseBatch) map[uint32]Owner {
	owners := make(map[uint32]Owner, len(now.Tiles)/2)
	for i := 0; i+1 < len(now.Tiles); i += 2 {
		tile := now.Start + uint32(i/2)
		country, seen := r.rewound[tile]
		if code := int(binary.LittleEndian.Uint16(now.Tiles[i:])); !seen && code < len(now.Codes) {
			country = now.Codes[code]
		}
		if country != "" {
			owners[tile] = Owner{Country: country}
		}
	}
	return owners
}

func (r *Rewind) Hits() map[uint32]uint32 {
	return r.hits
}
