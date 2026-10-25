package get_bombs_and_fortifies_handler

import (
	"context"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Query interface {
	BombsAndFortifies(ctx context.Context, accounts []ledger.AccountID) (*planetv1.GetBombsAndFortifiesResponse, error)
}

func New(query Query) GetBombsAndFortifiesHandler {
	return GetBombsAndFortifiesHandler{query: query}
}

type GetBombsAndFortifiesHandler struct {
	query Query
}

func (h GetBombsAndFortifiesHandler) GetBombsAndFortifies(
	ctx context.Context,
	req *connect.Request[planetv1.GetBombsAndFortifiesRequest],
) (*connect.Response[planetv1.GetBombsAndFortifiesResponse], error) {
	accounts := make([]ledger.AccountID, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		account, err := ledger.AccountIDOf(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		accounts = append(accounts, account)
	}

	answer, err := h.query.BombsAndFortifies(ctx, accounts)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers what is not the caller's fault.
	}
	return connect.NewResponse(answer), nil
}
