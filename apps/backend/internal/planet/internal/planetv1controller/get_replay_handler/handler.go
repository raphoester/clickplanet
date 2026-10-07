package get_replay_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/planetmessage"
)

type UseCase interface {
	Execute(ctx context.Context, in replay_usecase.In) (replay_usecase.Out, error)
}

func New(useCase UseCase) GetReplayHandler {
	return GetReplayHandler{useCase: useCase}
}

type GetReplayHandler struct {
	useCase UseCase
}

func (h GetReplayHandler) GetReplay(
	ctx context.Context,
	req *connect.Request[planetv1.GetReplayRequest],
) (*connect.Response[planetv1.GetReplayResponse], error) {
	var in replay_usecase.In
	if since := req.Msg.GetSince(); since != nil {
		in.Since = since.AsTime()
	}
	if until := req.Msg.GetUntil(); until != nil {
		in.Until = until.AsTime()
	}

	out, err := h.useCase.Execute(ctx, in)
	if errors.Is(err, replay_usecase.ErrInvalidWindow) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, err
	}

	events := make([]*planetv1.ReplayedEvent, len(out.Scenes))
	for i, scene := range out.Scenes {
		events[i] = &planetv1.ReplayedEvent{At: timestamppb.New(scene.At), Event: eventOf(scene)}
	}

	return connect.NewResponse(&planetv1.GetReplayResponse{
		Opening: planetmessage.Map(out.Opening),
		Events:  events,
		Since:   timestamppb.New(out.Since),
		Until:   timestamppb.New(out.Until),
	}), nil
}

func eventOf(scene ledger.Scene) *planetv1.PlanetEvent {
	switch {
	case scene.Blast != nil:
		return planetmessage.BombDropped(scene.Blast)
	case scene.Spread != nil:
		return planetmessage.TilesSpread(scene.Spread.Country, scene.Spread.Tile, scene.Spread.Tiles)
	case scene.Enclosure != nil:
		return planetmessage.TilesEnclosed(scene.Enclosure.Country, scene.Enclosure.Tile, nil, scene.Enclosure.Tiles, false)
	default:
		return planetmessage.TileUpdate(scene.Change.TileUpdate)
	}
}
