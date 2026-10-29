package set_name_shown_handler

import (
	"context"

	"connectrpc.com/connect"

	clipsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/clips/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/clipsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/optouts"
)

type UseCase interface {
	Execute(ctx context.Context, account optouts.AccountID, shown bool) error
}

func New(useCase UseCase) SetNameShownHandler {
	return SetNameShownHandler{useCase: useCase}
}

type SetNameShownHandler struct {
	useCase UseCase
}

func (h SetNameShownHandler) SetNameShown(
	ctx context.Context,
	req *connect.Request[clipsv1.SetNameShownRequest],
) (*connect.Response[clipsv1.SetNameShownResponse], error) {
	account, err := caller.AccountOf(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // already a connect error.
	}

	shown := req.Msg.GetNameShown()
	if err := h.useCase.Execute(ctx, account, shown); err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&clipsv1.SetNameShownResponse{NameShown: shown}), nil
}
