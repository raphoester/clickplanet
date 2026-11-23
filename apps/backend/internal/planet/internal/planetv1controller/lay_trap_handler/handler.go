package lay_trap_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetrefusal"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/traps/usecases/lay_trap_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in lay_trap_usecase.In) error
}

func New(useCase UseCase) LayTrapHandler {
	return LayTrapHandler{useCase: useCase}
}

type LayTrapHandler struct {
	useCase UseCase
}

func (h LayTrapHandler) LayTrap(
	ctx context.Context,
	req *connect.Request[planetv1.LayTrapRequest],
) (*connect.Response[planetv1.LayTrapResponse], error) {
	err := h.useCase.Execute(ctx, lay_trap_usecase.In{TileID: req.Msg.GetTileId(), CountryID: req.Msg.GetCountryId()})

	if refused, ok := planetrefusal.Refusal(err); ok {
		return nil, refused
	}
	if refused, ok := allegiancemessage.Refusal(err); ok {
		return nil, refused
	}

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.LayTrapResponse{}), nil
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, clicks.ErrTileOutOfRange):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, clicks.ErrNotTrappable):
		return nil, preconditionWith(clicks.ErrNotTrappable, &planetv1.TrapRefusal{})
	case errors.Is(err, lay_trap_usecase.ErrNoTrap):
		return nil, connect.NewError(connect.CodeNotFound, lay_trap_usecase.ErrNoTrap)
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
