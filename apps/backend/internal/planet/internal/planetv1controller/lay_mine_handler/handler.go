package lay_mine_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/mines/usecases/lay_mine_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/allegiancemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetrefusal"
)

type UseCase interface {
	Execute(ctx context.Context, in lay_mine_usecase.In) error
}

func New(useCase UseCase) LayMineHandler {
	return LayMineHandler{useCase: useCase}
}

type LayMineHandler struct {
	useCase UseCase
}

func (h LayMineHandler) LayMine(
	ctx context.Context,
	req *connect.Request[planetv1.LayMineRequest],
) (*connect.Response[planetv1.LayMineResponse], error) {
	err := h.useCase.Execute(ctx, lay_mine_usecase.In{TileID: req.Msg.GetTileId(), CountryID: req.Msg.GetCountryId()})

	if refused, ok := planetrefusal.Refusal(err); ok {
		return nil, refused
	}
	if refused, ok := allegiancemessage.Refusal(err); ok {
		return nil, refused
	}

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.LayMineResponse{}), nil
	case errors.Is(err, clicks.ErrUnknownCountry), errors.Is(err, clicks.ErrTileOutOfRange):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, clicks.ErrNotMineable):
		return nil, preconditionWith(clicks.ErrNotMineable, &planetv1.MineRefusal{})
	case errors.Is(err, lay_mine_usecase.ErrNoMine):
		return nil, connect.NewError(connect.CodeNotFound, lay_mine_usecase.ErrNoMine)
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
