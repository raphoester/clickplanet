package get_discord_users_handler

import (
	"context"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

// The most users one page holds, and what a limit of 0 asks.
const MaxLimit = 500

type Query interface {
	// The zero account asks from the first one.
	DiscordUsers(ctx context.Context, after accounts.AccountID, limit int) (*authv1.GetDiscordUsersResponse, error)
}

func New(query Query) GetDiscordUsersHandler {
	return GetDiscordUsersHandler{query: query}
}

type GetDiscordUsersHandler struct {
	query Query
}

func (h GetDiscordUsersHandler) GetDiscordUsers(
	ctx context.Context,
	req *connect.Request[authv1.GetDiscordUsersRequest],
) (*connect.Response[authv1.GetDiscordUsersResponse], error) {
	var after accounts.AccountID
	if value := req.Msg.GetAfterAccountId(); value != "" {
		account, err := accounts.AccountIDOf(value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		after = account
	}
	limit := int(req.Msg.GetLimit())
	if limit == 0 || limit > MaxLimit {
		limit = MaxLimit
	}

	found, err := h.query.DiscordUsers(ctx, after, limit)
	if err != nil {
		return nil, err //nolint:wrapcheck // the error net answers it.
	}
	return connect.NewResponse(found), nil
}
