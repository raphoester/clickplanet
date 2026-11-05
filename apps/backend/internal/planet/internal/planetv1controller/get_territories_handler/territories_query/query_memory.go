package territories_query

import (
	"cmp"
	"slices"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Tally interface {
	Tally() (clicks.MapVersion, clicks.Tally)
}

type Borders interface {
	CountryOfLandmass(landmass clicks.LandmassID) string
}

func NewMemoryQuery(tally Tally, borders Borders, tiles uint32) *MemoryQuery {
	return &MemoryQuery{tally: tally, borders: borders, tiles: tiles}
}

type MemoryQuery struct {
	tally   Tally
	borders Borders
	tiles   uint32
}

func (q *MemoryQuery) Territories() *planetv1.GetTerritoriesResponse {
	_, tally := q.tally.Tally()

	// A tile on no country's land counts for its flag and lies on no ground.
	grounds := make(map[string]map[string]uint32)
	for _, row := range tally.Landmasses {
		ground := q.borders.CountryOfLandmass(row.Landmass)
		if ground == "" {
			continue
		}
		for _, held := range row.Flags {
			if grounds[held.Flag] == nil {
				grounds[held.Flag] = make(map[string]uint32)
			}
			grounds[held.Flag][ground] += held.Tiles
		}
	}

	territories := make([]*planetv1.Territory, 0, len(tally.Countries))
	for _, count := range tally.Countries {
		territories = append(territories, &planetv1.Territory{
			CountryId: count.Flag, Tiles: count.Tiles, Grounds: groundsOf(grounds[count.Flag]),
		})
	}
	slices.SortFunc(territories, func(a, b *planetv1.Territory) int {
		return cmp.Compare(a.GetCountryId(), b.GetCountryId())
	})

	return &planetv1.GetTerritoriesResponse{Tiles: q.tiles, Territories: territories}
}

func groundsOf(held map[string]uint32) []*planetv1.Ground {
	grounds := make([]*planetv1.Ground, 0, len(held))
	for country, tiles := range held {
		grounds = append(grounds, &planetv1.Ground{CountryId: country, Tiles: tiles})
	}
	slices.SortFunc(grounds, func(a, b *planetv1.Ground) int {
		return cmp.Compare(a.GetCountryId(), b.GetCountryId())
	})
	return grounds
}
