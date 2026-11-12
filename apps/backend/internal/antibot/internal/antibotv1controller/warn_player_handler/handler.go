package warn_player_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/warn_player_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in warn_player_usecase.In) (warn_player_usecase.Out, error)
}

func New(useCase UseCase) WarnPlayerHandler {
	return WarnPlayerHandler{useCase: useCase}
}

type WarnPlayerHandler struct {
	useCase UseCase
}

func (h WarnPlayerHandler) WarnPlayer(
	ctx context.Context,
	req *connect.Request[antibotv1.WarnPlayerRequest],
) (*connect.Response[antibotv1.WarnPlayerResponse], error) {
	out, err := h.useCase.Execute(ctx, warn_player_usecase.In{Account: req.Msg.GetAccountId()})

	switch {
	case err == nil:
		return connect.NewResponse(&antibotv1.WarnPlayerResponse{
			AccountId:      out.Account,
			WarnedAt:       timestamppb.New(out.WarnedAt),
			AlreadyWarned:  out.AlreadyWarned,
			AcknowledgedAt: timestampOrNil(out.AcknowledgedAt),
		}), nil
	case errors.Is(err, bans.ErrInvalidAccount):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, warn_player_usecase.ErrNotEnforced):
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return nil, err
	}
}

func timestampOrNil(at time.Time) *timestamppb.Timestamp {
	if at.IsZero() {
		return nil
	}
	return timestamppb.New(at)
}
