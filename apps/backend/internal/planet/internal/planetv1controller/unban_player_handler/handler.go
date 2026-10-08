package unban_player_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/unban_player_usecase"
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
	req *connect.Request[planetv1.UnbanPlayerRequest],
) (*connect.Response[planetv1.UnbanPlayerResponse], error) {
	out, err := h.useCase.Execute(ctx, unban_player_usecase.In{Scope: req.Msg.GetScope(), Account: req.Msg.GetAccountId()})

	switch {
	case err == nil:
		return connect.NewResponse(&planetv1.UnbanPlayerResponse{
			Scope:       out.Scope,
			AccountId:   out.Account,
			Offence:     uint32(out.Offence), //nolint:gosec // an offence count, never negative.
			BannedUntil: timestamppb.New(out.Until),
		}), nil
	case errors.Is(err, ledger.ErrInvalidScope), errors.Is(err, ledger.ErrInvalidAccount), errors.Is(err, ledger.ErrNoCaller):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, unban_player_usecase.ErrNotBanned):
		return nil, connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, unban_player_usecase.ErrAntiBotOff):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return nil, err
	}
}
