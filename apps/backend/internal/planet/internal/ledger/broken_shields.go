package ledger

import (
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// brokenShields is each tile whose shields a revert's caller struck, on the map its revert reads.
type brokenShields struct {
	tiles map[uint32]broken
}

type broken struct {
	flag string
	// As the ledger last wrote them, and as they would stand had the caller never played.
	shields, without int
}

func newBrokenShields() *brokenShields {
	return &brokenShields{tiles: map[uint32]broken{}}
}

// record reads an event as the map with the caller's events reads it.
func (b *brokenShields) record(event Event, made bool) {
	struck := map[uint32]string{}
	if made {
		event.Strikes(func(strike Strike) { struck[strike.Tile] = strike.Owner })
	}

	marked(event, func(tile uint32, flag string, shields int, taken bool) {
		was, seen := b.tiles[tile]
		owner, hit := struck[tile]
		switch {
		case taken && !seen:
			return
		case taken && !made && flag != was.flag:
			delete(b.tiles, tile)
			return
		case taken:
			was.shields = 0
		case hit && (!seen || was.flag != owner):
			was = broken{flag: owner, shields: shields, without: shields + 1}
		case !seen:
			return
		case !made && flag == was.flag:
			was.without += shields - was.shields
			was.shields = shields
		default:
			was.shields = shields
		}
		b.tiles[tile] = was
	})
}

func (b *brokenShields) wipe() {
	clear(b.tiles)
}

func (b *brokenShields) cut(carry Carry) {
	before := b.tiles
	b.tiles = map[uint32]broken{}
	children := map[uint32][]uint32{}
	carry.Each(func(child, parent uint32) {
		if _, seen := before[parent]; seen {
			children[parent] = append(children[parent], child)
		}
	})
	for parent, of := range children {
		slices.Sort(of)
		was := before[parent]
		for rank, child := range of {
			b.tiles[child] = broken{
				flag:    was.flag,
				shields: clicks.ShareOf(was.shields, rank, len(of)),
				without: clicks.ShareOf(was.without, rank, len(of)),
			}
		}
	}
}

// marked visits each tile an event changed with the flag it shows after, the shields the ledger wrote on it, and
// whether it changed hands.
func marked(event Event, see func(tile uint32, flag string, shields int, taken bool)) {
	event.Show(func(scene Scene) {
		switch {
		case scene.Change != nil:
			change := scene.Change
			see(change.Tile, change.Value, change.Shields, change.Value != change.Previous)
		case scene.Blast != nil:
			for _, tile := range scene.Blast.Cleared {
				see(tile, "", 0, true)
			}
			for i, tile := range scene.Blast.Struck {
				see(tile, scene.Blast.StruckOwners[i], scene.Blast.Left[i], false)
			}
		case scene.Enclosure != nil:
			for _, impact := range scene.Enclosure.Impacts {
				switch impact.Outcome {
				case clicks.Taken:
					see(impact.Tile, scene.Enclosure.Country, 0, true)
				case clicks.Shielded:
					see(impact.Tile, impact.Owner, impact.Shields, false)
				case clicks.Unchanged:
				}
			}
		case scene.Spread != nil:
		}
	})
}
