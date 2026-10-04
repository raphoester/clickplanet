package cpcallers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type Auth interface {
	GetCaller(ctx context.Context, req *connect.Request[authv1.GetCallerRequest]) (*connect.Response[authv1.GetCallerResponse], error)
}

const askTimeout = time.Second

func New(auth Auth) *Callers {
	return &Callers{auth: auth}
}

type Callers struct {
	auth Auth
}

func (c *Callers) Caller(ctx context.Context, cookie string) (cpsession.AccountID, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := c.auth.GetCaller(ctx, connect.NewRequest(&authv1.GetCallerRequest{Cookie: cookie}))
	if err != nil {
		return cpsession.NoAccount, fmt.Errorf("failed to ask the auth module whose cookie this is: %w", err)
	}

	account, err := uuid.Parse(res.Msg.GetAccountId())
	if err != nil {
		return cpsession.NoAccount, nil //nolint:nilerr // an empty answer is a cookie that names nobody.
	}
	return cpsession.AccountID(account), nil
}

func NewLogged(inner *Callers, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  *Callers
	logger *slog.Logger
}

func (l *Logged) Caller(ctx context.Context, cookie string) (cpsession.AccountID, error) {
	account, err := l.inner.Caller(ctx, cookie)
	if err != nil {
		l.logger.Error("could not ask whose cookie a call was sent with", slog.Any("error", err))
	}
	return account, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
