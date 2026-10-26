package delete_entry_handler

import (
	"context"

	"connectrpc.com/connect"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/schedulemessage"
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
	req *connect.Request[seasonsv1.DeleteEntryRequest],
) (*connect.Response[seasonsv1.DeleteEntryResponse], error) {
	id, err := schedule.ParseEntryID(req.Msg.GetId())
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // NotFound.
	}
	if err := h.useCase.Execute(ctx, id); err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&seasonsv1.DeleteEntryResponse{}), nil
}
