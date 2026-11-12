package tiles_query_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/get_tiles_handler/tiles_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type stubHolds map[uint32]clicks.TileHold

func (s stubHolds) HoldsOf(tiles []uint32) []clicks.TileHold {
	holds := make([]clicks.TileHold, len(tiles))
	for i, tile := range tiles {
		holds[i] = s[tile]
		holds[i].Tile = tile
	}
	return holds
}

type stubGeography struct {
	neighbours map[uint32][]uint32
}

func (g stubGeography) Position(id uint32) (clicks.Vec3, bool) {
	return clicks.Vec3{X: float64(id), Y: 1, Z: -1}, true
}

func (g stubGeography) Neighbours(id uint32) []uint32 { return g.neighbours[id] }

func TestEachTileIsItsFlagItsHolderItsPlaceAndWhatItTouches(t *testing.T) {
	ana := cpsession.AccountID(uuid.MustParse("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"))
	table := []uint32{3, 5}
	query := tiles_query.NewMemoryQuery(
		stubHolds{4: {Flag: "fr", Holder: ana}, 9: {Flag: "de", Holder: cpsession.NoAccount}},
		stubGeography{neighbours: map[uint32][]uint32{4: table}},
	)

	answer := query.Tiles([]uint32{4, 9})
	table[0] = 99

	assert.True(t, proto.Equal(&planetv1.GetTilesResponse{Tiles: []*planetv1.MapTile{
		{
			TileId: 4, CountryId: "fr", HolderAccountId: ana.String(), Point: &planetv1.GlobePoint{X: 4, Y: 1, Z: -1},
			Neighbours: []uint32{3, 5},
		},
		{TileId: 9, CountryId: "de", Point: &planetv1.GlobePoint{X: 9, Y: 1, Z: -1}},
	}}, answer), "a tile nobody holds names no account, and the answer is a copy of the map's table: %v", answer)
}
