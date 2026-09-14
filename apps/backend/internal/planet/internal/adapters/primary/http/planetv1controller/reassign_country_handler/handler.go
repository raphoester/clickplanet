package reassign_country_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country"
)

type UseCase interface {
	Execute(ctx context.Context, in reassign_country.In) (reassign_country.Out, error)
}

func New(useCase UseCase) ReassignCountryHandler {
	return ReassignCountryHandler{useCase: useCase}
}

type ReassignCountryHandler struct {
	useCase UseCase
}

func (h ReassignCountryHandler) ReassignCountry(
	ctx context.Context,
	req *connect.Request[planetv1.ReassignCountryRequest],
) (*connect.Response[planetv1.ReassignCountryResponse], error) {
	out, err := h.useCase.Execute(ctx, reassign_country.In{
		From:   req.Msg.GetFromCountryId(),
		To:     req.Msg.GetToCountryId(),
		DryRun: req.Msg.GetDryRun(),
	})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.ReassignCountryResponse{
			FromBefore: uint32(out.FromBefore), //nolint:gosec // a tile count, bounded by the map.
			ToBefore:   uint32(out.ToBefore),   //nolint:gosec // a tile count, bounded by the map.
			Moved:      uint32(out.Moved),      //nolint:gosec // a tile count, bounded by the map.
			FromAfter:  uint32(out.FromAfter),  //nolint:gosec // a tile count, bounded by the map.
			ToAfter:    uint32(out.ToAfter),    //nolint:gosec // a tile count, bounded by the map.
		}), nil
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, reassign_country.ErrSameCountry):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
