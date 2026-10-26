package add_entry_handler

import (
	"context"

	"connectrpc.com/connect"

	rulesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/rules/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/rulesv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
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
	req *connect.Request[rulesv1.AddEntryRequest],
) (*connect.Response[rulesv1.AddEntryResponse], error) {
	id, err := h.useCase.Execute(ctx, schedulemessage.DraftOf(
		req.Msg.GetRuleSet(), req.Msg.GetStartsAt(), req.Msg.GetEndsAt(), req.Msg.GetPriority()))
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&rulesv1.AddEntryResponse{Id: id.String()}), nil
}
