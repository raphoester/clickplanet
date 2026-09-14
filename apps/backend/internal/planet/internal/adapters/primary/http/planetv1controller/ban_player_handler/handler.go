package ban_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/ban_player"
)

type UseCase interface {
	Execute(ctx context.Context, in ban_player.In) (ban_player.Out, error)
}

func New(useCase UseCase) BanPlayerHandler {
	return BanPlayerHandler{useCase: useCase}
}

type BanPlayerHandler struct {
	useCase UseCase
}

func (h BanPlayerHandler) BanPlayer(
	ctx context.Context,
	req *connect.Request[planetv1.BanPlayerRequest],
) (*connect.Response[planetv1.BanPlayerResponse], error) {
	in := ban_player.In{Scope: req.Msg.GetScope()}
	if duration := req.Msg.GetDuration(); duration != nil {
		in.Duration = duration.AsDuration()
	}

	out, err := h.useCase.Execute(ctx, in)

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.BanPlayerResponse{
			Scope:       out.Scope,
			Offence:     uint32(out.Offence), //nolint:gosec // an offence count, never negative.
			BannedUntil: timestamppb.New(out.Until),
			Enforced:    out.Enforced,
		}), nil
	case errors.Is(err, clicks.ErrInvalidScope), errors.Is(err, ban_player.ErrNegativeDuration):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ban_player.ErrAntiBotOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return nil, err
	}
}
