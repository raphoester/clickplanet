package territories_query

import (
	"cmp"
	"slices"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
)

type Territories interface {
	Territories() map[string]uint32
}

func NewMemoryQuery(territories Territories, tiles uint32) *MemoryQuery {
	return &MemoryQuery{territories: territories, tiles: tiles}
}

type MemoryQuery struct {
	territories Territories
	tiles       uint32
}

func (q *MemoryQuery) Territories() *planetv1.GetTerritoriesResponse {
	held := q.territories.Territories()

	territories := make([]*planetv1.Territory, 0, len(held))
	for country, tiles := range held {
		territories = append(territories, &planetv1.Territory{CountryId: country, Tiles: tiles})
	}
	slices.SortFunc(territories, func(a, b *planetv1.Territory) int {
		return cmp.Compare(a.GetCountryId(), b.GetCountryId())
	})

	return &planetv1.GetTerritoriesResponse{Tiles: q.tiles, Territories: territories}
}
