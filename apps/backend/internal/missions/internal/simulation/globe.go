package simulation

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

var ErrNoGround = errors.New("a tile is on no country's land")

// The map replayed from a blank one: what each tile wears, and what each flag holds on each country's land.
type Globe struct {
	grounds []string
	sizes   []missions.Holding
	owners  []string
	held    map[string]map[string]uint32
	// Acts whose tile wore another flag than the replay says: a reassign, a paint or a revert the ledger never saw.
	drift uint64
	// Acts on a tile past the map.
	outside uint64
}

// grounds is the country of each tile's land, the tile ids from 1.
func NewGlobe(grounds []string) (*Globe, error) {
	sizes := make(map[string]uint32)
	for n, ground := range grounds {
		if ground == "" {
			return nil, fmt.Errorf("%w: tile %d", ErrNoGround, n+1)
		}
		sizes[ground]++
	}
	globe := &Globe{grounds: grounds, owners: make([]string, len(grounds)), held: make(map[string]map[string]uint32)}
	for _, country := range slices.Sorted(maps.Keys(sizes)) {
		globe.sizes = append(globe.sizes, missions.Holding{Country: country, Tiles: sizes[country]})
	}
	return globe, nil
}

// Paints the tile in flag, "" for none: previous is the flag the ledger says it wore.
func (g *Globe) Paint(tile uint32, flag, previous string) {
	if tile == 0 || int(tile) > len(g.owners) {
		g.outside++
		return
	}
	at := tile - 1
	if g.owners[at] != previous {
		g.drift++
	}
	if worn := g.owners[at]; worn != "" {
		g.held[worn][g.grounds[at]]--
	}
	g.owners[at] = flag
	if flag == "" {
		return
	}
	if g.held[flag] == nil {
		g.held[flag] = make(map[string]uint32)
	}
	g.held[flag][g.grounds[at]]++
}

// The country whose land the tile is on, "" past the map.
func (g *Globe) GroundOf(tile uint32) string {
	if tile == 0 || int(tile) > len(g.grounds) {
		return ""
	}
	return g.grounds[tile-1]
}

// What each flag holds now, as planet tells missions, flags and grounds by code.
func (g *Globe) World() missions.World {
	world := missions.World{CountrySizes: slices.Clone(g.sizes)}
	for _, flag := range slices.Sorted(maps.Keys(g.held)) {
		territory := missions.Territory{Flag: flag}
		for _, ground := range slices.Sorted(maps.Keys(g.held[flag])) {
			if tiles := g.held[flag][ground]; tiles > 0 {
				territory.Tiles += tiles
				territory.Grounds = append(territory.Grounds, missions.Holding{Country: ground, Tiles: tiles})
			}
		}
		if territory.Tiles > 0 {
			world.Territories = append(world.Territories, territory)
		}
	}
	slices.SortStableFunc(world.Territories, func(a, b missions.Territory) int { return cmp.Compare(b.Tiles, a.Tiles) })
	return world
}

// The part of the map the flag holds now.
func (g *Globe) Share(flag string) float64 {
	var tiles uint32
	for _, held := range g.held[flag] {
		tiles += held
	}
	return float64(tiles) / float64(len(g.grounds))
}
