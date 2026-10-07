package ledger

import (
	"cmp"
	"encoding/binary"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Scenes from since until until, and what each tile held at since: the first scene after since that touches it says.
func NewFootage(since, until time.Time) *Footage {
	return &Footage{since: since, until: until, owners: make(map[uint32]string), shields: make(map[uint32]int)}
}

type Footage struct {
	since   time.Time
	until   time.Time
	scenes  []Scene
	owners  map[uint32]string
	shields map[uint32]int
}

func (f *Footage) See(event Event) {
	event.Show(f.see)
}

func (f *Footage) see(scene Scene) {
	if scene.At.Before(f.since) {
		return
	}

	switch {
	case scene.Change != nil:
		f.rewind(scene.Change.Tile, scene.Change.Previous)
		f.rewindShields(scene.Change.Tile, scene.Change.Was)
	case scene.Blast != nil:
		for i, tile := range scene.Blast.Cleared {
			f.rewind(tile, scene.Blast.Owners[i])
			f.rewindShields(tile, 0)
		}
		for i, tile := range scene.Blast.Struck {
			f.rewindShields(tile, scene.Blast.Left[i]+1)
		}
	}

	if scene.At.Before(f.until) {
		f.scenes = append(f.scenes, scene)
	}
}

func (f *Footage) rewind(tile uint32, owner string) {
	if _, seen := f.owners[tile]; !seen {
		f.owners[tile] = owner
	}
}

func (f *Footage) rewindShields(tile uint32, shields int) {
	if _, seen := f.shields[tile]; !seen {
		f.shields[tile] = shields
	}
}

func (f *Footage) Scenes() []Scene {
	return slices.Clone(f.scenes)
}

// The map as it was at since, from the map now.
func (f *Footage) Opening(now clicks.DenseBatch) clicks.DenseBatch {
	opening := clicks.DenseBatch{Start: now.Start, Codes: slices.Clone(now.Codes), Tiles: slices.Clone(now.Tiles)}

	codes := make(map[string]uint16, len(opening.Codes))
	for i, code := range opening.Codes {
		codes[code] = uint16(i) //nolint:gosec // the storage interns at most a uint16 of codes.
	}

	end := now.Start + uint32(len(now.Tiles)/2) //nolint:gosec // two bytes per tile of a uint32 range.
	within := func(tile uint32) bool { return tile >= now.Start && tile < end }

	for tile, owner := range f.owners {
		if !within(tile) {
			continue
		}
		code, known := codes[owner]
		if !known {
			code = uint16(len(opening.Codes)) //nolint:gosec // as above.
			codes[owner] = code
			opening.Codes = append(opening.Codes, owner)
		}
		binary.LittleEndian.PutUint16(opening.Tiles[(tile-now.Start)*2:], code)
	}

	shields := make(map[uint32]int, len(now.Shields)+len(f.shields))
	for _, tile := range now.Shields {
		shields[tile.Tile] = tile.Shields
	}
	for tile, count := range f.shields {
		if within(tile) {
			shields[tile] = count
		}
	}
	for tile, count := range shields {
		if count > 0 {
			opening.Shields = append(opening.Shields, clicks.TileShields{Tile: tile, Shields: count})
		}
	}
	slices.SortFunc(opening.Shields, func(a, b clicks.TileShields) int { return cmp.Compare(a.Tile, b.Tile) })

	return opening
}
