// Package create_session_usecase admits a caller: it checks Turnstile, brings back the account its cookie holds or starts a guest, and mints the click token for that account.
package create_session_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sessions interface {
	FindSession(ctx context.Context, tokenHash accounts.TokenHash) (*accounts.Session, error)
	CreateGuest(ctx context.Context, session *accounts.Session) error
	SaveSession(ctx context.Context, session *accounts.Session) error
}

type Minter interface {
	Mint(ip string, account uuid.UUID, now time.Time) (*cpsession.Token, error)
}

type In struct {
	AttestationToken string
	IP               string
	CookieHeader     string
}

// Out is the click token and the Set-Cookie to send back (empty when the cookie needs no change).
type Out struct {
	Token     *cpsession.Token
	Account   accounts.AccountID
	SetCookie string
}

type UseCase struct {
	attester attestation.Attester
	sessions Sessions
	ids      accounts.IDProvider
	tokens   accounts.TokenGenerator
	minter   Minter
	lifetime accounts.Lifetime
	clock    cptime.Clock
}

func New(
	attester attestation.Attester,
	sessions Sessions,
	ids accounts.IDProvider,
	tokens accounts.TokenGenerator,
	minter Minter,
	lifetime accounts.Lifetime,
	clock cptime.Clock,
) *UseCase {
	return &UseCase{
		attester: attester,
		sessions: sessions,
		ids:      ids,
		tokens:   tokens,
		minter:   minter,
		lifetime: lifetime.WithDefaults(),
		clock:    clock,
	}
}

// Execute answers attestation.ErrAttestationFailed for a caller that proved nothing.
func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	// Without an address the token binds to the empty string, which every other caller would verify against too.
	if in.IP == "" {
		return nil, fmt.Errorf("%w: the request carries no source address", attestation.ErrAttestationFailed)
	}
	if err := u.attester.Attest(ctx, in.AttestationToken, in.IP); err != nil {
		return nil, fmt.Errorf("%w: %w", attestation.ErrAttestationFailed, err)
	}

	now := u.clock.Now()

	admitted, err := u.resume(ctx, in.CookieHeader, now)
	if ended(err) {
		admitted, err = u.startGuest(ctx, now)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller's account: %w", err)
	}

	token, err := u.minter.Mint(in.IP, uuid.UUID(admitted.Account), now)
	if err != nil {
		return nil, fmt.Errorf("failed to mint the click token: %w", err)
	}

	admitted.Token = token
	return admitted, nil
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

// ended is a caller with no session to resume: no cookie, an unknown token, or an expired session.
func ended(err error) bool {
	return errors.Is(err, accounts.ErrNoSessionCookie) ||
		errors.Is(err, accounts.ErrSessionNotFound) ||
		errors.Is(err, accounts.ErrSessionExpired)
}
