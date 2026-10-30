package unmute_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/unmute_usecase"
)

func New(useCase unmute_usecase.Executor) UnmuteHandler {
	return UnmuteHandler{useCase: useCase}
}

type UnmuteHandler struct {
	useCase unmute_usecase.Executor
}

func (h UnmuteHandler) Unmute(
	ctx context.Context,
	req *connect.Request[chatv1.UnmuteRequest],
) (*connect.Response[chatv1.UnmuteResponse], error) {
	until, err := h.useCase.Execute(ctx, messages.AccountIDOf(req.Msg.GetAccountId()))

	switch {
	case err == nil:
		return connect.NewResponse(&chatv1.UnmuteResponse{MutedUntil: timestamppb.New(until)}), nil
	case errors.Is(err, mutes.ErrNoAccount):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, mutes.ErrNotMuted):
		return nil, connect.NewError(connect.CodeNotFound, err)
	default:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
}
