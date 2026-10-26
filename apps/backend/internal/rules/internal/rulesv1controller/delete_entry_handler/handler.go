package delete_entry_handler

import (
	"context"

	"connectrpc.com/connect"

	rulesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/rules/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/rulesv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/rules/internal/schedule"
)

type UseCase interface {
	Execute(ctx context.Context, id schedule.EntryID) error
}

func New(useCase UseCase) DeleteEntryHandler {
	return DeleteEntryHandler{useCase: useCase}
}

type DeleteEntryHandler struct {
	useCase UseCase
}

func (h DeleteEntryHandler) DeleteEntry(
	ctx context.Context,
	req *connect.Request[rulesv1.DeleteEntryRequest],
) (*connect.Response[rulesv1.DeleteEntryResponse], error) {
	id, err := schedule.ParseEntryID(req.Msg.GetId())
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // NotFound.
	}
	if err := h.useCase.Execute(ctx, id); err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&rulesv1.DeleteEntryResponse{}), nil
}
