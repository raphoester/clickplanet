package accounts

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type SessionFinder interface {
	Session(ctx context.Context, tokenHash TokenHash) (*Session, error)
}

// Caller is the live session a browser's cookie holds. Absent is ErrNoAccount; a store failure is not.
func Caller(ctx context.Context, sessions SessionFinder, cookieHeader string, now time.Time) (*Session, error) {
	token, err := TokenFromCookies(cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoAccount, err)
	}

	session, err := sessions.Session(ctx, token.Hash)
	if errors.Is(err, ErrSessionNotFound) {
		return nil, fmt.Errorf("%w: %w", ErrNoAccount, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the session: %w", err)
	}

	if err := session.ExpiryError(now); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNoAccount, err)
	}
	return session, nil
}

type AccountFinder interface {
	Account(ctx context.Context, account AccountID) (*Account, error)
}
