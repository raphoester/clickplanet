package sign_out_usecase

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type Sessions interface {
	Session(ctx context.Context, tokenHash accounts.TokenHash) (*accounts.Session, error)
	DeleteSession(ctx context.Context, tokenHash accounts.TokenHash) error
}

type Publisher interface {
	Publish(event proto.Message)
}

type UseCase struct {
	sessions Sessions
	events   Publisher
}

func New(sessions Sessions, events Publisher) *UseCase {
	return &UseCase{sessions: sessions, events: events}
}

func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (string, error) {
	token, err := accounts.TokenFromCookies(cookieHeader)
	if errors.Is(err, accounts.ErrNoSessionCookie) {
		return accounts.ExpiredSessionCookie(), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the cookie: %w", err)
	}

	session, err := u.sessions.Session(ctx, token.Hash)
	if errors.Is(err, accounts.ErrSessionNotFound) {
		return accounts.ExpiredSessionCookie(), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to find the session: %w", err)
	}

	if err := u.sessions.DeleteSession(ctx, token.Hash); err != nil {
		return "", fmt.Errorf("failed to delete the session: %w", err)
	}

	u.events.Publish(&authv1.SignedOut{AccountId: session.Account.String()})

	return accounts.ExpiredSessionCookie(), nil
}
