package top_players_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/top_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/adminplayer"
)

type UseCase interface {
	Execute(ctx context.Context, in top_players_usecase.In) (top_players_usecase.Out, error)
}

func New(useCase UseCase) TopPlayersHandler {
	return TopPlayersHandler{useCase: useCase}
}

type TopPlayersHandler struct {
	useCase UseCase
}

func (h TopPlayersHandler) TopPlayers(
	ctx context.Context,
	req *connect.Request[planetv1.TopPlayersRequest],
) (*connect.Response[planetv1.TopPlayersResponse], error) {
	out, err := h.useCase.Execute(ctx, top_players_usecase.In{Limit: int(req.Msg.GetLimit())})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&planetv1.TopPlayersResponse{
		Players: adminplayer.Encode(out.Players),
		Total:   uint32(out.Total), //nolint:gosec // a scope count, bounded by the map.
	}), nil
}
