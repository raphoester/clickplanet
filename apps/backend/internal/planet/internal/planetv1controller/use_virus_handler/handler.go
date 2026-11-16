package use_virus_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetrefusal"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/viruses/usecases/use_virus_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in use_virus_usecase.In) error
}

func New(useCase UseCase) UseVirusHandler {
	return UseVirusHandler{useCase: useCase}
}

type UseVirusHandler struct {
	useCase UseCase
}

func (h UseVirusHandler) UseVirus(
	ctx context.Context,
	req *connect.Request[planetv1.UseVirusRequest],
) (*connect.Response[planetv1.UseVirusResponse], error) {
	err := h.useCase.Execute(ctx, use_virus_usecase.In{TileID: req.Msg.GetTileId(), CountryID: req.Msg.GetCountryId()})

	if refused, ok := planetrefusal.Refusal(err); ok {
		return nil, refused
	}
	if refused, ok := allegiancemessage.Refusal(err); ok {
		return nil, refused
	}

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.UseVirusResponse{}), nil
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, clicks.ErrTileOutOfRange):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, clicks.ErrOutOfReach):
		return nil, preconditionWith(clicks.ErrOutOfReach, &planetv1.ReachRefusal{})
	case errors.Is(err, clicks.ErrNotInfectable):
		return nil, preconditionWith(clicks.ErrNotInfectable, &planetv1.VirusRefusal{})
	case errors.Is(err, use_virus_usecase.ErrNoVirus):
		return nil, connect.NewError(connect.CodeNotFound, use_virus_usecase.ErrNoVirus)
	default:
		return nil, err
	}
}

func preconditionWith(err error, detail proto.Message) *connect.Error {
	refused := connect.NewError(connect.CodeFailedPrecondition, err)
	if detail, detailErr := connect.NewErrorDetail(detail); detailErr == nil {
		refused.AddDetail(detail)
	}
	return refused
}
