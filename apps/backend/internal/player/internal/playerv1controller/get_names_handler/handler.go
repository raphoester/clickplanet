// Package get_names_handler serves player.v1.InternalService/GetNames, for the other modules.
package get_names_handler

import (
	"context"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type UseCase interface {
	Execute(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Name, error)
}

func New(useCase UseCase) GetNamesHandler {
	return GetNamesHandler{useCase: useCase}
}

type GetNamesHandler struct {
	useCase UseCase
}

// GetNames leaves out an id that is not an account, as it leaves out an account with no name.
func (h GetNamesHandler) GetNames(
	ctx context.Context,
	req *connect.Request[playerv1.GetNamesRequest],
) (*connect.Response[playerv1.GetNamesResponse], error) {
	accounts := make([]players.AccountID, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		if account, err := players.AccountIDOf(id); err == nil {
			accounts = append(accounts, account)
		}
	}

	names, err := h.useCase.Execute(ctx, accounts)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}

	answer := make(map[string]string, len(names))
	for account, name := range names {
		answer[account.String()] = string(name)
	}

	return connect.NewResponse(&playerv1.GetNamesResponse{Names: answer}), nil
}
