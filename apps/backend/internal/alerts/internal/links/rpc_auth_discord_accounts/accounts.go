package rpc_auth_discord_accounts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type Auth interface {
	GetDiscordUsers(
		ctx context.Context,
		req *connect.Request[authv1.GetDiscordUsersRequest],
	) (*connect.Response[authv1.GetDiscordUsersResponse], error)
}

const askTimeout = 5 * time.Second

func New(auth Auth) Accounts {
	return Accounts{auth: auth}
}

type Accounts struct {
	auth Auth
}

// The accounts signed in with Discord after this one, in the order of their ids; the zero account asks from the first.
// A page shorter than the limit is the last.
func (a Accounts) DiscordAccounts(ctx context.Context, after links.AccountID, limit int) ([]links.DiscordAccount, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	req := &authv1.GetDiscordUsersRequest{Limit: uint32(limit)} //nolint:gosec // a page size, far below 2^32.
	if after != (links.AccountID{}) {
		req.AfterAccountId = after.String()
	}
	res, err := a.auth.GetDiscordUsers(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("failed to ask auth who signed in with Discord: %w", err)
	}

	page := make([]links.DiscordAccount, 0, len(res.Msg.GetUsers()))
	for _, signed := range res.Msg.GetUsers() {
		account, err := links.AccountIDOf(signed.GetAccountId())
		if err != nil {
			return nil, fmt.Errorf("auth named an account that is none: %w", err)
		}
		user, err := discord.UserIDOf(signed.GetDiscordUserId())
		if err != nil {
			return nil, fmt.Errorf("auth named a Discord user that is none for %s: %w", account, err)
		}
		page = append(page, links.DiscordAccount{Account: account, User: user})
	}
	return page, nil
}
