package rpc_auth_grants

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/links"
)

type Auth interface {
	GetDiscordGrant(
		ctx context.Context,
		req *connect.Request[authv1.GetDiscordGrantRequest],
	) (*connect.Response[authv1.GetDiscordGrantResponse], error)
}

var ErrNoToken = errors.New("auth answered no access token to join the server with")

const askTimeout = 2 * time.Second

func New(auth Auth) Grants {
	return Grants{auth: auth}
}

type Grants struct {
	auth Auth
}

// links.ErrNoGrant when auth keeps it no more. The token is in no error.
func (g Grants) Grant(ctx context.Context, id links.GrantID) (links.Grant, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := g.auth.GetDiscordGrant(ctx, connect.NewRequest(&authv1.GetDiscordGrantRequest{GrantId: string(id)}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return links.Grant{}, fmt.Errorf("%w: %s", links.ErrNoGrant, id)
	}
	if err != nil {
		return links.Grant{}, fmt.Errorf("failed to ask auth for the grant %s: %w", id, err)
	}
	account, err := links.AccountIDOf(res.Msg.GetAccountId())
	if err != nil {
		return links.Grant{}, fmt.Errorf("auth named no account for the grant %s: %w", id, err)
	}
	user, err := discord.UserIDOf(res.Msg.GetDiscordUserId())
	if err != nil {
		return links.Grant{}, fmt.Errorf("auth named no Discord user for the grant %s: %w", id, err)
	}
	if res.Msg.GetAccessToken() == "" {
		return links.Grant{}, fmt.Errorf("%w: %s", ErrNoToken, id)
	}
	return links.Grant{Account: account, User: user, Token: discord.AccessToken(res.Msg.GetAccessToken())}, nil
}
