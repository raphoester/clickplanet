package clicks

import (
	"maps"
	"slices"
)

// Capital is the tile a country starts an arena on.
type Capital struct {
	Country string
	Tile    uint32
}

// StartTile is a tile an arena's map starts with: a capital, held by its country, with its shields.
type StartTile struct {
	Tile    uint32
	Country string
	Shields int
}

// Capitals is each country's capital tile.
type Capitals struct {
	tiles     map[string]uint32
	countries map[uint32]string
}

func NewCapitals(tiles map[string]uint32) *Capitals {
	countries := make(map[uint32]string, len(tiles))
	for country, tile := range tiles {
		countries[tile] = country
	}

	return &Capitals{tiles: maps.Clone(tiles), countries: countries}
}

// Flags is the capitals of the countries a player can play for: ground no flag stands for has none.
func (c *Capitals) Flags(flag func(country string) bool) *Capitals {
	kept := make(map[string]uint32, len(c.tiles))
	for country, tile := range c.tiles {
		if flag(country) {
			kept[country] = tile
		}
	}

	return NewCapitals(kept)
}

func (c *Capitals) Is(tile uint32) bool {
	_, ok := c.countries[tile]
	return ok
}

// All is every capital, by country.
func (c *Capitals) All() []Capital {
	all := make([]Capital, 0, len(c.tiles))
	for _, country := range slices.Sorted(maps.Keys(c.tiles)) {
		all = append(all, Capital{Country: country, Tile: c.tiles[country]})
	}

	return all
}
