package ban_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/ban_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in ban_player_usecase.In) (ban_player_usecase.Out, error)
}

func New(useCase UseCase) BanPlayerHandler {
	return BanPlayerHandler{useCase: useCase}
}

type BanPlayerHandler struct {
	useCase UseCase
}

func (h BanPlayerHandler) BanPlayer(
	ctx context.Context,
	req *connect.Request[antibotv1.BanPlayerRequest],
) (*connect.Response[antibotv1.BanPlayerResponse], error) {
	in := ban_player_usecase.In{Scope: req.Msg.GetScope(), Account: req.Msg.GetAccountId()}
	if duration := req.Msg.GetDuration(); duration != nil {
		in.Duration = duration.AsDuration()
	}

	out, err := h.useCase.Execute(ctx, in)

	switch {
	case err == nil:
		return connect.NewResponse(&antibotv1.BanPlayerResponse{
			Scope:       out.Scope,
			AccountId:   out.Account,
			Offence:     uint32(out.Offence), //nolint:gosec // an offence count, never negative.
			BannedUntil: timestamppb.New(out.Until),
			Enforced:    out.Enforced,
		}), nil
	case errors.Is(err, bans.ErrInvalidScope), errors.Is(err, bans.ErrInvalidAccount), errors.Is(err, bans.ErrNoCaller),
		errors.Is(err, ban_player_usecase.ErrNegativeDuration):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err
	}
}
