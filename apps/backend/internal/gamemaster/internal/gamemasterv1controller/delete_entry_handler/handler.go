package delete_entry_handler

import (
	"context"

	"connectrpc.com/connect"

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/schedulemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
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
	req *connect.Request[gamemasterv1.DeleteEntryRequest],
) (*connect.Response[gamemasterv1.DeleteEntryResponse], error) {
	id, err := schedule.ParseEntryID(req.Msg.GetId())
	if err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // NotFound.
	}
	if err := h.useCase.Execute(ctx, id); err != nil {
		return nil, schedulemessage.Refusal(err) //nolint:wrapcheck // the error net answers what is not the operator's.
	}
	return connect.NewResponse(&gamemasterv1.DeleteEntryResponse{}), nil
}
