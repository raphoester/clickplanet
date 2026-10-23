package get_audience_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

type Audience interface {
	Audience() bonuses.Audience
}

func New(audience Audience) GetAudienceHandler {
	return GetAudienceHandler{audience: audience}
}

type GetAudienceHandler struct {
	audience Audience
}

func (h GetAudienceHandler) GetAudience(
	_ context.Context,
	_ *connect.Request[planetv1.GetAudienceRequest],
) (*connect.Response[planetv1.GetAudienceResponse], error) {
	audience := h.audience.Audience()

	return connect.NewResponse(&planetv1.GetAudienceResponse{
		Streams:  uint32(audience.Streams),  //nolint:gosec // the streams open on one server.
		Entrants: uint32(audience.Entrants), //nolint:gosec // as above.
	}), nil
}
