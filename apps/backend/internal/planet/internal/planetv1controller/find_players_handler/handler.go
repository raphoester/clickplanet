package find_players_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/find_players_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/adminplayer"
)

type UseCase interface {
	Execute(ctx context.Context, in find_players_usecase.In) (find_players_usecase.Out, error)
}

func New(useCase UseCase) FindPlayersHandler {
	return FindPlayersHandler{useCase: useCase}
}

type FindPlayersHandler struct {
	useCase UseCase
}

func (h FindPlayersHandler) FindPlayers(
	ctx context.Context,
	req *connect.Request[planetv1.FindPlayersRequest],
) (*connect.Response[planetv1.FindPlayersResponse], error) {
	out, err := h.useCase.Execute(ctx, find_players_usecase.In{
		Flag:  req.Msg.GetFlagCountryId(),
		Area:  req.Msg.GetAreaCountryId(),
		Limit: int(req.Msg.GetLimit()),
	})

	switch {
	case err == nil:
	case errors.Is(err, clicks.ErrUnknownCountry):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}

	return connect.NewResponse(&planetv1.FindPlayersResponse{
		Players: adminplayer.Encode(out.Players),
		Total:   uint32(out.Total), //nolint:gosec // a scope count, bounded by the map.
	}), nil
}
