package get_accounts_handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Accounts interface {
	Accounts(ctx context.Context, accounts []players.AccountID) (map[players.AccountID]players.Account, error)
}

func New(accounts Accounts) GetAccountsHandler {
	return GetAccountsHandler{accounts: accounts}
}

type GetAccountsHandler struct {
	accounts Accounts
}

func (h GetAccountsHandler) GetAccounts(
	ctx context.Context,
	req *connect.Request[playerv1.GetAccountsRequest],
) (*connect.Response[playerv1.GetAccountsResponse], error) {
	asked := make([]players.AccountID, 0, len(req.Msg.GetAccountIds()))
	for _, id := range req.Msg.GetAccountIds() {
		account, err := players.AccountIDOf(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		asked = append(asked, account)
	}

	known, err := h.accounts.Accounts(ctx, asked)
	if err != nil {
		return nil, fmt.Errorf("failed to read the accounts: %w", err)
	}

	answer := &playerv1.GetAccountsResponse{Accounts: make([]*playerv1.Account, 0, len(known))}
	for _, id := range asked {
		account, ok := known[id]
		if !ok {
			continue
		}
		delete(known, id)
		answer.Accounts = append(answer.Accounts, accountOf(id, account))
	}
	return connect.NewResponse(answer), nil
}

func accountOf(id players.AccountID, account players.Account) *playerv1.Account {
	encoded := &playerv1.Account{AccountId: id.String(), Linked: account.Linked()}
	if !account.CreatedAt().IsZero() {
		encoded.CreatedAtUnixMs = account.CreatedAt().UnixMilli()
	}
	return encoded
}
