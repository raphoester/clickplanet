package mute_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatoperator"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/proto_kind"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/usecases/mute_usecase"
)

func New(useCase mute_usecase.Executor) MuteHandler {
	return MuteHandler{useCase: useCase}
}

type MuteHandler struct {
	useCase mute_usecase.Executor
}

func (h MuteHandler) Mute(
	ctx context.Context,
	req *connect.Request[chatv1.MuteRequest],
) (*connect.Response[chatv1.MuteResponse], error) {
	kind, err := proto_kind.Decode(req.Msg.GetKind())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	in := mute_usecase.In{
		Account: messages.AccountIDOf(req.Msg.GetAccountId()), Kind: kind, Reason: req.Msg.GetReason(),
		By: chatoperator.Of(ctx), AccountOnly: req.Msg.GetAccountOnly(),
	}
	if duration := req.Msg.GetDuration(); duration != nil {
		in.Duration = duration.AsDuration()
	}

	mute, err := h.useCase.Execute(ctx, in)

	switch {
	case err == nil:
		return connect.NewResponse(&chatv1.MuteResponse{
			Scope:      string(mute.Caller().Scope()),
			MutedUntil: timestamppb.New(mute.Until()),
		}), nil
	case errors.Is(err, mutes.ErrNoAccount), errors.Is(err, mutes.ErrInvalidDuration),
		errors.Is(err, moderation.ErrInvalidReason):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
}
