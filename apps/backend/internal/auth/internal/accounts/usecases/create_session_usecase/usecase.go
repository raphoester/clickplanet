package create_session_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Sessions interface {
	Session(ctx context.Context, tokenHash accounts.TokenHash) (*accounts.Session, error)
	CreateGuest(ctx context.Context, session *accounts.Session) error
	SaveSession(ctx context.Context, session *accounts.Session) error
}

type Minter interface {
	Mint(ip string, holder cpsession.Holder, now time.Time) (*cpsession.Token, error)
}

type In struct {
	AttestationToken string
	IP               string
	CookieHeader     string
}

type Out struct {
	Token     *cpsession.Token
	Account   accounts.AccountID
	Linked    bool
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

func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	// An empty IP binds the token to "", which every other caller would verify against too.
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

	token, err := u.minter.Mint(in.IP, cpsession.Holder{Account: admitted.Account, Linked: admitted.Linked}, now)
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

	session, err := u.sessions.Session(ctx, token.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to find the session: %w", err)
	}
	if err := session.ExpiryError(now); err != nil {
		return nil, fmt.Errorf("failed to resume the session: %w", err)
	}

	if !session.Extendable(now, u.lifetime) {
		return &Out{Account: session.Account, Linked: session.Linked}, nil
	}

	extended := session.Extended(now, u.lifetime)
	if err := u.sessions.SaveSession(ctx, extended); err != nil {
		return nil, fmt.Errorf("failed to save the extended session: %w", err)
	}

	return &Out{Account: extended.Account, Linked: extended.Linked, SetCookie: extended.Cookie(token, now)}, nil
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

	session := accounts.GuestSession(account, token, u.lifetime, now)
	if err := u.sessions.CreateGuest(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to store the guest: %w", err)
	}

	return &Out{Account: session.Account, SetCookie: session.Cookie(token, now)}, nil
}

func ended(err error) bool {
	return errors.Is(err, accounts.ErrNoSessionCookie) ||
		errors.Is(err, accounts.ErrSessionNotFound) ||
		errors.Is(err, accounts.ErrSessionExpired)
}
