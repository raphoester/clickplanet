package get_tile_holder_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type TilesChecker interface {
	CheckTile(tile uint32) bool
}

type Query interface {
	TileHolder(ctx context.Context, tile uint32) (*planetv1.GetTileHolderResponse, error)
}

func New(tiles TilesChecker, query Query) GetTileHolderHandler {
	return GetTileHolderHandler{tiles: tiles, query: query}
}

type GetTileHolderHandler struct {
	tiles TilesChecker
	query Query
}

func (h GetTileHolderHandler) GetTileHolder(
	ctx context.Context,
	req *connect.Request[planetv1.GetTileHolderRequest],
) (*connect.Response[planetv1.GetTileHolderResponse], error) {
	tile := req.Msg.GetTileId()
	if !h.tiles.CheckTile(tile) {
		return nil, connect.NewError(connect.CodeInvalidArgument, clicks.ErrTileOutOfRange)
	}
	holder, err := h.query.TileHolder(ctx, tile)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(holder), nil
}
