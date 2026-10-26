package unban_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/unban_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in unban_player_usecase.In) (unban_player_usecase.Out, error)
}

func New(useCase UseCase) UnbanPlayerHandler {
	return UnbanPlayerHandler{useCase: useCase}
}

type UnbanPlayerHandler struct {
	useCase UseCase
}

func (h UnbanPlayerHandler) UnbanPlayer(
	ctx context.Context,
	req *connect.Request[antibotv1.UnbanPlayerRequest],
) (*connect.Response[antibotv1.UnbanPlayerResponse], error) {
	out, err := h.useCase.Execute(ctx, unban_player_usecase.In{Scope: req.Msg.GetScope(), Account: req.Msg.GetAccountId()})

	switch {
	case err == nil:
		return connect.NewResponse(&antibotv1.UnbanPlayerResponse{
			Scope:         out.Scope,
			AccountId:     out.Account,
			Offence:       uint32(out.Offence), //nolint:gosec // an offence count, never negative.
			BannedUntil:   timestamppb.New(out.Until),
			PardonedUntil: timestamppb.New(out.PardonedUntil),
		}), nil
	case errors.Is(err, bans.ErrInvalidScope), errors.Is(err, bans.ErrInvalidAccount), errors.Is(err, bans.ErrNoCaller):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, unban_player_usecase.ErrNotBanned):
		return nil, connect.NewError(connect.CodeNotFound, err)
	default:
		return nil, err
	}
}
