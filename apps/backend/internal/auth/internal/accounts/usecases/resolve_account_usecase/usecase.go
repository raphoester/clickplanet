// Package resolve_account_usecase finds the account a browser's cookie belongs to, and gives it a guest one when asked.
package resolve_account_usecase

import (
	"context"
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

// Out is the account (uuid.Nil for none) and the Set-Cookie to send back (empty for none).
type Out struct {
	Account   uuid.UUID
	SetCookie string
}

func New(sessions accounts.Sessions, lifetime accounts.Lifetime, clock cptime.Clock) *UseCase {
	return &UseCase{sessions: sessions, lifetime: lifetime.WithDefaults(), clock: clock}
}

type UseCase struct {
	sessions accounts.Sessions
	lifetime accounts.Lifetime
	clock    cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	now := u.clock.Now()

	if token, ok := accounts.TokenFrom(in.CookieHeader); ok {
		out, found, err := u.resume(ctx, token, now)
		if err != nil || found {
			return out, err
		}
	}

	if !in.Create {
		return Out{}, nil
	}

	return u.createGuest(ctx, now)
}

func (u *UseCase) resume(ctx context.Context, token string, now time.Time) (Out, bool, error) {
	hash := accounts.HashOf(token)

	session, found, err := u.sessions.FindSession(ctx, hash)
	if err != nil {
		return Out{}, false, err //nolint:wrapcheck // the store names what failed.
	}
	if !found || !session.Live(now) {
		return Out{}, false, nil
	}

	if !u.lifetime.ExtensionDue(session, now) {
		return Out{Account: session.Account}, true, nil
	}

	expiresAt := u.lifetime.ExpiryFrom(now)
	if err := u.sessions.ExtendSession(ctx, hash, expiresAt, now); err != nil {
		return Out{}, false, err //nolint:wrapcheck // the store names what failed.
	}

	return Out{Account: session.Account, SetCookie: accounts.SetCookie(token, expiresAt, now)}, true, nil
}

func (u *UseCase) createGuest(ctx context.Context, now time.Time) (Out, error) {
	account, err := uuid.NewV7()
	if err != nil {
		return Out{}, fmt.Errorf("failed to generate an account id: %w", err)
	}

	token, err := accounts.NewToken()
	if err != nil {
		return Out{}, fmt.Errorf("failed to generate a session token: %w", err)
	}

	expiresAt := u.lifetime.ExpiryFrom(now)
	if err := u.sessions.CreateGuest(ctx, account, token.Hash, expiresAt, now); err != nil {
		return Out{}, err //nolint:wrapcheck // the store names what failed.
	}

	return Out{Account: account, SetCookie: accounts.SetCookie(token.Value, expiresAt, now)}, nil
}
