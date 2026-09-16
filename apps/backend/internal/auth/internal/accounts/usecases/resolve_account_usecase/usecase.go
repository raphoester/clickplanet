// Package resolve_account_usecase finds the account a browser's cookie belongs to, and gives it a guest one when asked.
package resolve_account_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	CookieHeader string
	Create       bool
}

// Out is the account and the Set-Cookie to send back (empty for none).
type Out struct {
	Account   uuid.UUID
	SetCookie string
}

type UseCase struct {
	sessions accounts.Sessions
	ids      accounts.IDProvider
	tokens   accounts.TokenGenerator
	lifetime accounts.Lifetime
	clock    cptime.Clock
}

func New(
	sessions accounts.Sessions,
	ids accounts.IDProvider,
	tokens accounts.TokenGenerator,
	lifetime accounts.Lifetime,
	clock cptime.Clock,
) *UseCase {
	return &UseCase{sessions: sessions, ids: ids, tokens: tokens, lifetime: lifetime.WithDefaults(), clock: clock}
}

// Execute answers accounts.ErrNoAccount when the browser has no live session and none was asked for.
func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	now := u.clock.Now()

	out, err := u.resume(ctx, in.CookieHeader, now)
	if err == nil {
		return out, nil
	}
	if !ended(err) {
		return nil, fmt.Errorf("failed to resume the session: %w", err)
	}

	if !in.Create {
		return nil, fmt.Errorf("%w: %w", accounts.ErrNoAccount, err)
	}

	out, err = u.startGuest(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("failed to start a guest: %w", err)
	}
	return out, nil
}

func (u *UseCase) resume(ctx context.Context, cookieHeader string, now time.Time) (*Out, error) {
	token, err := accounts.TokenFromCookies(cookieHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to read the cookie: %w", err)
	}

	session, err := u.sessions.FindSession(ctx, token.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to find the session: %w", err)
	}
	if err := session.CheckLive(now); err != nil {
		return nil, fmt.Errorf("failed to resume the session: %w", err)
	}

	if !session.ExtendIfDue(now, u.lifetime) {
		return &Out{Account: session.Account}, nil
	}
	if err := u.sessions.SaveSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save the extended session: %w", err)
	}

	return &Out{Account: session.Account, SetCookie: session.Cookie(token, now)}, nil
}

func (u *UseCase) startGuest(ctx context.Context, now time.Time) (*Out, error) {
	account, err := u.ids.NewID()
	if err != nil {
		return nil, fmt.Errorf("failed to get an account id: %w", err)
	}
	token, err := u.tokens.NewToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get a session token: %w", err)
	}

	session := accounts.StartGuest(account, token, u.lifetime, now)
	if err := u.sessions.CreateGuest(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store the guest: %w", err)
	}

	return &Out{Account: session.Account, SetCookie: session.Cookie(token, now)}, nil
}

// ended is a browser with no session to resume: no cookie, an unknown token, or an expired session.
func ended(err error) bool {
	return errors.Is(err, accounts.ErrNoSessionCookie) ||
		errors.Is(err, accounts.ErrSessionNotFound) ||
		errors.Is(err, accounts.ErrSessionExpired)
}
