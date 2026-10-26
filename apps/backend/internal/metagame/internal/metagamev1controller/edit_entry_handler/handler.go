package edit_entry_handler

import (
	"context"

	"connectrpc.com/connect"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
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
	req *connect.Request[metagamev1.EditEntryRequest],
) (*connect.Response[metagamev1.EditEntryResponse], error) {
	id, err := schedule.ParseEntryID(req.Msg.GetId())
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // NotFound.
	}
	if err := h.useCase.Execute(ctx, id, schedulemessage.DraftOf(
		req.Msg.GetRuleSet(), req.Msg.GetStartsAt(), req.Msg.GetEndsAt(), req.Msg.GetPriority())); err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&metagamev1.EditEntryResponse{}), nil
}
