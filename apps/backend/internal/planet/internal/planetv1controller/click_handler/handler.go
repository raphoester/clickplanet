package click_handler

import (
	"context"

	"connectrpc.com/connect"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/clickbudget"
)

type UseCase interface {
	Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error)
}

func New(useCase UseCase) ClickHandler {
	return ClickHandler{useCase: useCase}
}

type ClickHandler struct {
	useCase UseCase
}

func (h ClickHandler) Click(
	ctx context.Context,
	req *connect.Request[planetv1.ClickRequest],
) (*connect.Response[planetv1.ClickResponse], error) {
	out, err := h.useCase.Execute(ctx, click_usecase.In{
		TileID:    req.Msg.GetTileId(),
		CountryID: req.Msg.GetCountryId(),
		Spread:    req.Msg.GetSpread(),
		Enclose:   req.Msg.GetEnclose(),
	})
	if err != nil {
		return nil, toConnect(err, out)
	}

	res := &planetv1.ClickResponse{Gift: out.Gift}
	if out.Limited {
		res.Budget = clickbudget.Encode(out.Budget)
	}

	return connect.NewResponse(res), nil
}
