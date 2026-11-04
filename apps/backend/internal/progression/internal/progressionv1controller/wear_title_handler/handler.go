package wear_title_handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/progressionmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing"
)

type UseCase interface {
	Execute(ctx context.Context, account accounts.AccountID, title titles.ID) (titles.Standing, error)
}

func New(useCase UseCase) WearTitleHandler {
	return WearTitleHandler{useCase: useCase}
}

type WearTitleHandler struct {
	useCase UseCase
}

func (h WearTitleHandler) WearTitle(
	ctx context.Context,
	req *connect.Request[progressionv1.WearTitleRequest],
) (*connect.Response[progressionv1.WearTitleResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already the connect error the caller reads.
	}

	worn, err := h.useCase.Execute(ctx, account, titles.ID(req.Msg.GetTitleId()))
	switch {
	case errors.Is(err, wearing.ErrNotWearable):
		return nil, connect.NewError(connect.CodeInvalidArgument, wearing.ErrNotWearable)
	case err != nil:
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}

	return connect.NewResponse(&progressionv1.WearTitleResponse{Worn: progressionmessage.Title(worn)}), nil
}
