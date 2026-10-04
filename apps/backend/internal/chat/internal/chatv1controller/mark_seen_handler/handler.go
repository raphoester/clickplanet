package mark_seen_handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/mark_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type UseCase interface {
	Execute(ctx context.Context, in mark_seen_usecase.In) error
}

func New(useCase UseCase) MarkSeenHandler {
	return MarkSeenHandler{useCase: useCase}
}

type MarkSeenHandler struct {
	useCase UseCase
}

func (h MarkSeenHandler) MarkSeen(
	ctx context.Context,
	req *connect.Request[chatv1.MarkSeenRequest],
) (*connect.Response[chatv1.MarkSeenResponse], error) {
	err := h.useCase.Execute(ctx, mark_seen_usecase.In{
		Account: messages.AccountIDOf(cpctx.GetAccount(ctx)),
		At:      time.UnixMilli(req.Msg.GetSeenUntilUnixMs()).UTC(),
	})
	switch {
	case errors.Is(err, messages.ErrNoAccount):
		return nil, connect.NewError(connect.CodeUnauthenticated, messages.ErrNoAccount)
	case errors.Is(err, seen.ErrNoTime):
		return nil, connect.NewError(connect.CodeInvalidArgument, seen.ErrNoTime)
	case err != nil:
		return nil, err //nolint:wrapcheck // a storage failure is the error net's to answer.
	}

	return connect.NewResponse(&chatv1.MarkSeenResponse{}), nil
}
