package arenas

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"

// Starts is how an arena's map starts: each flag's capital, held by it, with the shields the rules in force give it.
type Starts struct {
	capitals *clicks.Capitals
	rules    func() Rules
}

func NewStarts(capitals *clicks.Capitals, rules func() Rules) Starts {
	return Starts{capitals: capitals, rules: rules}
}

func (s Starts) Tiles() []clicks.StartTile {
	shields := s.rules().CapitalShields
	capitals := s.capitals.All()
	tiles := make([]clicks.StartTile, 0, len(capitals))
	for _, capital := range capitals {
		tiles = append(tiles, clicks.StartTile{Tile: capital.Tile, Country: capital.Country, Shields: shields})
	}

	return tiles
}
