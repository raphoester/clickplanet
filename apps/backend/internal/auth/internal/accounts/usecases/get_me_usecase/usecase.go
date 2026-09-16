// Package get_me_usecase reads the account a browser's cookie belongs to. It creates, extends and saves nothing.
package get_me_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type SessionFinder interface {
	FindSession(ctx context.Context, tokenHash []byte) (*accounts.Session, error)
}

type UseCase struct {
	sessions SessionFinder
	clock    cptime.Clock
}

func New(sessions SessionFinder, clock cptime.Clock) *UseCase {
	return &UseCase{sessions: sessions, clock: clock}
}

// Execute answers accounts.ErrNoAccount when the cookie holds no live session.
func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (uuid.UUID, error) {
	token, err := accounts.TokenFromCookies(cookieHeader)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %w", accounts.ErrNoAccount, err)
	}

	session, err := u.sessions.FindSession(ctx, token.Hash)
	if errors.Is(err, accounts.ErrSessionNotFound) {
		return uuid.Nil, fmt.Errorf("%w: %w", accounts.ErrNoAccount, err)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to find the session: %w", err)
	}

	if err := session.CheckLive(u.clock.Now()); err != nil {
		return uuid.Nil, fmt.Errorf("%w: %w", accounts.ErrNoAccount, err)
	}

	return session.Account, nil
}
