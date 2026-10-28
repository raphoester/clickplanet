package add_entry_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type UseCase interface {
	Execute(ctx context.Context, draft schedule.Draft) (schedule.EntryID, error)
}

func New(useCase UseCase) AddEntryHandler {
	return AddEntryHandler{useCase: useCase}
}

type AddEntryHandler struct {
	useCase UseCase
}

func (h AddEntryHandler) AddEntry(
	ctx context.Context,
	req *connect.Request[gamemasterv1.AddEntryRequest],
) (*connect.Response[gamemasterv1.AddEntryResponse], error) {
	draft, err := schedulemessage.DraftOf(req.Msg.GetRuleSet(), req.Msg.GetKind(), req.Msg.GetDescription(),
		req.Msg.GetStartsAt(), req.Msg.GetEndsAt(), req.Msg.GetPriority(), req.Msg.GetTimes())
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // InvalidArgument.
	}
	id, err := h.useCase.Execute(ctx, draft)
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&gamemasterv1.AddEntryResponse{Id: id.String()}), nil
}
