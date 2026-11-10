package seeds

import (
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
)

// Capitals starts each country on its capital, held by it, with Shields shields.
type Capitals struct {
	Shields int
}

func (c Capitals) Validate() error {
	return rules.AtMost("capitals.shields", c.Shields, "what a tile keeps", math.MaxUint8) //nolint:wrapcheck // the bound names the value.
}

func (c Capitals) Seed(ground Ground) arenas.Seed {
	return capitalSeed{capitals: ground.Capitals, shields: c.Shields}
}

type capitalSeed struct {
	capitals *clicks.Capitals
	shields  int
}

func (s capitalSeed) Tiles() []clicks.StartTile {
	capitals := s.capitals.All()
	tiles := make([]clicks.StartTile, 0, len(capitals))
	for _, capital := range capitals {
		tiles = append(tiles, clicks.StartTile{Tile: capital.Tile, Country: capital.Country, Shields: s.shields})
	}

	return tiles
}
