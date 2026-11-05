package edit_entry_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type UseCase interface {
	Execute(ctx context.Context, id schedule.EntryID, draft schedule.Draft) error
}

func New(useCase UseCase) EditEntryHandler {
	return EditEntryHandler{useCase: useCase}
}

type EditEntryHandler struct {
	useCase UseCase
}

func (h EditEntryHandler) EditEntry(
	ctx context.Context,
	req *connect.Request[gamemasterv1.EditEntryRequest],
) (*connect.Response[gamemasterv1.EditEntryResponse], error) {
	id, err := schedule.ParseEntryID(req.Msg.GetId())
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // NotFound.
	}
	draft, err := schedulemessage.DraftOf(req.Msg.GetRuleSet(), req.Msg.GetKind(), req.Msg.GetDescription(),
		req.Msg.GetStartsAt(), req.Msg.GetEndsAt(), req.Msg.GetPriority(), req.Msg.GetTimes())
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // InvalidArgument.
	}
	if err := h.useCase.Execute(ctx, id, draft); err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&gamemasterv1.EditEntryResponse{}), nil
}
