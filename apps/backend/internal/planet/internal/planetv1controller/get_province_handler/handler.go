package get_province_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var errProvinceOutOfRange = errors.New("province id out of range")

// Board is the board in play: its provinces.
type Board interface {
	Provinces() *clicks.Provinces
}

type Query interface {
	Province(ctx context.Context, province clicks.ProvinceID) (*planetv1.GetProvinceResponse, error)
}

func New(board Board, query Query) GetProvinceHandler {
	return GetProvinceHandler{board: board, query: query}
}

type GetProvinceHandler struct {
	board Board
	query Query
}

func (h GetProvinceHandler) GetProvince(
	ctx context.Context,
	req *connect.Request[planetv1.GetProvinceRequest],
) (*connect.Response[planetv1.GetProvinceResponse], error) {
	province := req.Msg.GetProvinceId()
	if province == uint32(clicks.NoProvince) || int(province) > h.board.Provinces().Count() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errProvinceOutOfRange)
	}
	answer, err := h.query.Province(ctx, clicks.ProvinceID(province)) //nolint:gosec // under the provinces' count, a uint16.
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(answer), nil
}
