package rpc_planet_map_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/rpc_planet_map"
)

const ana = "01900000-0000-7000-8000-000000000001"

type planet struct {
	asked [][]uint32
	short bool
}

func (p *planet) GetTiles(_ context.Context, req *connect.Request[planetv1.GetTilesRequest]) (*connect.Response[planetv1.GetTilesResponse], error) {
	p.asked = append(p.asked, req.Msg.GetTileIds())
	var answer []*planetv1.MapTile
	for _, id := range req.Msg.GetTileIds() {
		answer = append(answer, &planetv1.MapTile{
			TileId: id, CountryId: "ro", HolderAccountId: ana,
			Point: &planetv1.GlobePoint{X: 1, Y: 2, Z: 3}, Neighbours: []uint32{id + 1},
		})
	}
	if p.short {
		answer = answer[1:]
	}
	return connect.NewResponse(&planetv1.GetTilesResponse{Tiles: answer}), nil
}

func TestEachTileIsReadInTheMapsWords(t *testing.T) {
	tiles, err := rpc_planet_map.New(&planet{}).Tiles(t.Context(), []drawings.TileID{7})

	require.NoError(t, err)
	assert.Equal(t, []drawings.MapTile{{
		Tile:   drawings.Tile{ID: 7, Point: drawings.Point{X: 1, Y: 2, Z: 3}, Neighbours: []drawings.TileID{8}},
		Flag:   "ro",
		Holder: cpsession.AccountID(uuid.MustParse(ana)),
	}}, tiles)
}

func TestManyTilesAreAskedInBatchesPlanetTakes(t *testing.T) {
	ids := make([]drawings.TileID, 25_000)
	for i := range ids {
		ids[i] = drawings.TileID(i + 1) //nolint:gosec // a few thousand.
	}
	asked := &planet{}

	tiles, err := rpc_planet_map.New(asked).Tiles(t.Context(), ids)

	require.NoError(t, err)
	assert.Len(t, tiles, 25_000)
	require.Len(t, asked.asked, 3)
	assert.Len(t, asked.asked[0], 10_000)
	assert.Len(t, asked.asked[2], 5_000)
}

func TestAnAnswerShortOfATileIsAnError(t *testing.T) {
	_, err := rpc_planet_map.New(&planet{short: true}).Tiles(t.Context(), []drawings.TileID{7, 8})

	assert.ErrorContains(t, err, "planet answered 1 tiles of the 2 asked")
}
