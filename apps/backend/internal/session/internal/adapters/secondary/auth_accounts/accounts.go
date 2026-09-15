// Package auth_accounts asks the auth module who a browser is, over the internal listener.
package auth_accounts

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1/authv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
)

type Config struct {
	// Off mints every token with no account and never calls the auth module.
	Enabled bool

	// How long a mint waits on the auth module (default 2s).
	Timeout time.Duration
}

const defaultTimeout = 2 * time.Second

func (c Config) WithDefaults() Config {
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
	return c
}

type Accounts struct {
	client  authv1connect.InternalServiceClient
	timeout time.Duration
}

var _ domain.Accounts = (*Accounts)(nil)

func New(httpClient connect.HTTPClient, baseURL string, config Config) *Accounts {
	return &Accounts{
		client:  authv1connect.NewInternalServiceClient(httpClient, baseURL),
		timeout: config.WithDefaults().Timeout,
	}
}

func (a *Accounts) Resolve(ctx context.Context, cookieHeader string, create bool) (domain.Resolution, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	res, err := a.client.ResolveAccount(ctx, connect.NewRequest(&authv1.ResolveAccountRequest{
		CookieHeader: cookieHeader,
		Create:       create,
	}))
	if err != nil {
		return domain.Resolution{}, fmt.Errorf("the auth module did not resolve the account: %w", err)
	}

	account := uuid.Nil
	if id := res.Msg.GetAccountId(); id != "" {
		if account, err = uuid.Parse(id); err != nil {
			return domain.Resolution{}, fmt.Errorf("the auth module answered an account id that is not a uuid: %w", err)
		}
	}

	return domain.Resolution{Account: account, SetCookie: res.Msg.GetSetCookie()}, nil
}
