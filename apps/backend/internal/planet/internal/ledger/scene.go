package ledger

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// One of Change, Blast, Spread and Enclosure is set: what every screen was shown, as the live stream showed it.
type Scene struct {
	At        time.Time
	Change    *Change
	Blast     *clicks.Blast
	Spread    *Bonus
	Enclosure *Bonus
}

type Change struct {
	clicks.TileUpdate

	// The tile's shields before the change.
	Was int
}

type Bonus struct {
	Tile    uint32
	Country string
	Tiles   []uint32
}

func showImpacts(see func(Scene), at time.Time, flag string, impacts []clicks.Impact) {
	for _, impact := range impacts {
		switch impact.Outcome {
		case clicks.Taken:
			see(Scene{At: at, Change: &Change{
				TileUpdate: clicks.TileUpdate{Tile: impact.Tile, Value: flag, Previous: impact.Owner},
			}})
		case clicks.Shielded:
			see(Scene{At: at, Change: struck(impact.Tile, impact.Owner, impact.Shields, false)})
		case clicks.Unchanged:
		}
	}
}

func struck(tile uint32, owner string, shields int, clicked bool) *Change {
	return &Change{
		TileUpdate: clicks.TileUpdate{Tile: tile, Value: owner, Previous: owner, Clicked: clicked, Shields: shields},
		Was:        shields + 1,
	}
}

func bonusAt(tile uint32, country string, impacts []clicks.Impact) *Bonus {
	tiles := make([]uint32, len(impacts))
	for i, impact := range impacts {
		tiles[i] = impact.Tile
	}

	return &Bonus{Tile: tile, Country: country, Tiles: tiles}
}
