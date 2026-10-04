package name_accounts_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_accounts_usecase"
)

func New(useCase name_accounts_usecase.Executor) NameAccountsHandler {
	return NameAccountsHandler{useCase: useCase}
}

type NameAccountsHandler struct {
	useCase name_accounts_usecase.Executor
}

func (h NameAccountsHandler) NameAccounts(
	ctx context.Context,
	_ *connect.Request[playerv1.NameAccountsRequest],
) (*connect.Response[playerv1.NameAccountsResponse], error) {
	named, err := h.useCase.Execute(ctx)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(&playerv1.NameAccountsResponse{
		Named: uint32(named), //nolint:gosec // counted from accounts named, never negative or past 2^32.
	}), nil
}
