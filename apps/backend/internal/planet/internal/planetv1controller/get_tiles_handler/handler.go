package get_tiles_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The vandal squad asks the tiles its players painted in the last hour once a minute: a few thousand.
const MaxTiles = 20_000

var ErrTooManyTiles = fmt.Errorf("more than %d tiles asked at once", MaxTiles)

type TilesChecker interface {
	CheckTile(tile uint32) bool
}

type Query interface {
	Tiles(tiles []uint32) *planetv1.GetTilesResponse
}

func New(tiles TilesChecker, query Query) GetTilesHandler {
	return GetTilesHandler{tiles: tiles, query: query}
}

type GetTilesHandler struct {
	tiles TilesChecker
	query Query
}

func (h GetTilesHandler) GetTiles(
	_ context.Context,
	req *connect.Request[planetv1.GetTilesRequest],
) (*connect.Response[planetv1.GetTilesResponse], error) {
	asked := req.Msg.GetTileIds()
	if len(asked) > MaxTiles {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrTooManyTiles)
	}
	for _, tile := range asked {
		if !h.tiles.CheckTile(tile) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("tile %d: %w", tile, clicks.ErrTileOutOfRange))
		}
	}

	return connect.NewResponse(h.query.Tiles(asked)), nil
}
