package find_players_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/find_players_usecase"
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

	players := make([]*planetv1.Player, 0, len(out.Players))
	for _, player := range out.Players {
		players = append(players, &planetv1.Player{
			Scope:       player.Scope,
			Tiles:       uint32(player.Tiles), //nolint:gosec // a tile count, bounded by the map.
			FirstAt:     timestamppb.New(player.FirstAt),
			LastAt:      timestamppb.New(player.LastAt),
			Banned:      player.Banned,
			BannedUntil: timestampOrNil(player.BannedUntil),
			Offence:     uint32(player.Offence), //nolint:gosec // an offence count, never negative.
		})
	}

	return connect.NewResponse(&planetv1.FindPlayersResponse{
		Players: players,
		Total:   uint32(out.Total), //nolint:gosec // a scope count, bounded by the map.
	}), nil
}

func timestampOrNil(at time.Time) *timestamppb.Timestamp {
	if at.IsZero() {
		return nil
	}
	return timestamppb.New(at)
}
