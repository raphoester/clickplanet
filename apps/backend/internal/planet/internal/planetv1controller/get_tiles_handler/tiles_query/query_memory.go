package tiles_query

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Holds interface {
	HoldsOf(tiles []uint32) []clicks.TileHold
}

type Geography interface {
	Position(id uint32) (clicks.Vec3, bool)
	Neighbours(id uint32) []uint32
}

func NewMemoryQuery(holds Holds, geography Geography) *MemoryQuery {
	return &MemoryQuery{holds: holds, geography: geography}
}

type MemoryQuery struct {
	holds     Holds
	geography Geography
}

// The flags and holders under one read of the tiles' lock, then the map's shape, which never changes.
func (q *MemoryQuery) Tiles(tiles []uint32) *planetv1.GetTilesResponse {
	holds := q.holds.HoldsOf(tiles)

	answer := &planetv1.GetTilesResponse{Tiles: make([]*planetv1.MapTile, len(holds))}
	for i, hold := range holds {
		tile := &planetv1.MapTile{TileId: hold.Tile, CountryId: hold.Flag}
		if hold.Holder != cpsession.NoAccount {
			tile.HolderAccountId = hold.Holder.String()
		}
		if point, ok := q.geography.Position(hold.Tile); ok {
			tile.Point = &planetv1.GlobePoint{X: point.X, Y: point.Y, Z: point.Z}
		}
		// A copy: the geography's slice is the map's own table.
		tile.Neighbours = append([]uint32(nil), q.geography.Neighbours(hold.Tile)...)
		answer.Tiles[i] = tile
	}

	return answer
}
