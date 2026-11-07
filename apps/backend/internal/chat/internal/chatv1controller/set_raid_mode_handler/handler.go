package set_raid_mode_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/chat/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/chatv1controller/chatoperator"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/moderation"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/usecases/set_raid_mode_usecase"
)

func New(useCase set_raid_mode_usecase.Executor) SetRaidModeHandler {
	return SetRaidModeHandler{useCase: useCase}
}

type SetRaidModeHandler struct {
	useCase set_raid_mode_usecase.Executor
}

func (h SetRaidModeHandler) SetRaidMode(
	ctx context.Context,
	req *connect.Request[chatv1.SetRaidModeRequest],
) (*connect.Response[chatv1.SetRaidModeResponse], error) {
	in := set_raid_mode_usecase.In{Reason: req.Msg.GetReason(), By: chatoperator.Of(ctx)}
	if duration := req.Msg.GetDuration(); duration != nil {
		in.Duration = duration.AsDuration()
	}

	mode, running, err := h.useCase.Execute(ctx, in)

	switch {
	case err == nil && running:
		return connect.NewResponse(&chatv1.SetRaidModeResponse{RaidMode: Encode(mode)}), nil
	case err == nil:
		return connect.NewResponse(&chatv1.SetRaidModeResponse{}), nil
	case errors.Is(err, raids.ErrInvalidDuration), errors.Is(err, moderation.ErrInvalidReason):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
}

func Encode(mode raids.RaidMode) *chatv1.RaidMode {
	snapshot := mode.TakeSnapshot()
	return &chatv1.RaidMode{
		StartedAt: timestamppb.New(snapshot.StartedAt),
		EndsAt:    timestamppb.New(snapshot.EndsAt),
		Reason:    string(snapshot.Reason),
	}
}
