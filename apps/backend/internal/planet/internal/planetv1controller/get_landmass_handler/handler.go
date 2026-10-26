package get_landmass_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Borders interface {
	Landmasses() int
}

type Query interface {
	Landmass(ctx context.Context, landmass clicks.LandmassID) (*planetv1.GetLandmassResponse, error)
}

func New(borders Borders, query Query) GetLandmassHandler {
	return GetLandmassHandler{borders: borders, query: query}
}

type GetLandmassHandler struct {
	borders Borders
	query   Query
}

func (h GetLandmassHandler) GetLandmass(
	ctx context.Context,
	req *connect.Request[planetv1.GetLandmassRequest],
) (*connect.Response[planetv1.GetLandmassResponse], error) {
	landmass := req.Msg.GetLandmassId()
	if landmass == uint32(clicks.NoLandmass) || int(landmass) >= h.borders.Landmasses() {
		return nil, connect.NewError(connect.CodeInvalidArgument, clicks.ErrLandmassOutOfRange)
	}
	answer, err := h.query.Landmass(ctx, clicks.LandmassID(landmass)) //nolint:gosec // under the borders' count, a uint16.
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(answer), nil
}
