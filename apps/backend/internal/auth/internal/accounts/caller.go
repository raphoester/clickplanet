package accounts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SessionFinder interface {
	FindSession(ctx context.Context, tokenHash []byte) (*Session, error)
}

// Caller is the live session a browser's cookie holds. Absent is ErrNoAccount; a store failure is not.
func Caller(ctx context.Context, sessions SessionFinder, cookieHeader string, now time.Time) (*Session, error) {
	token, err := TokenFromCookies(cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoAccount, err)
	}

	session, err := sessions.FindSession(ctx, token.Hash)
	if errors.Is(err, ErrSessionNotFound) {
		return nil, fmt.Errorf("%w: %w", ErrNoAccount, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the session: %w", err)
	}

	if err := session.CheckLive(now); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoAccount, err)
	}
	return session, nil
}

type AccountFinder interface {
	FindAccount(ctx context.Context, account uuid.UUID) (*Account, error)
}
