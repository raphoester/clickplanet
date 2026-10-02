// Package start_email_sign_in_usecase opens an email sign-in: a code, sent to the address, and sealed in the browser's cookie.
package start_email_sign_in_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Limiter interface {
	Take(key string) (bool, cpratelimit.State)
}

type In struct {
	Address          string
	Intent           accounts.Intent
	AttestationToken string
	IP               string
	CookieHeader     string
}

type Out struct {
	SetCookie string
}

type UseCase struct {
	offered   bool
	attester  attestation.Attester
	blocklist signin.Blocklist
	sessions  accounts.SessionFinder
	sends     Limiter
	secrets   signin.Secrets
	codes     signin.Codes
	sealer    signin.ChallengeSealer
	mailer    signin.Mailer
	clock     cptime.Clock
}

func New(
	offered bool,
	attester attestation.Attester,
	blocklist signin.Blocklist,
	sessions accounts.SessionFinder,
	sends Limiter,
	secrets signin.Secrets,
	codes signin.Codes,
	sealer signin.ChallengeSealer,
	mailer signin.Mailer,
	clock cptime.Clock,
) *UseCase {
	return &UseCase{
		offered:   offered,
		attester:  attester,
		blocklist: blocklist,
		sessions:  sessions,
		sends:     sends,
		secrets:   secrets,
		codes:     codes,
		sealer:    sealer,
		mailer:    mailer,
		clock:     clock,
	}
}

// Execute answers signin.ErrSignInOff, signin.ErrAddressInvalid, signin.ErrAddressDisposable, attestation.ErrAttestationFailed,
// accounts.ErrNoAccount for a link from a browser with no account, and signin.ErrTooManyCodes, having sent nothing.
func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	if !u.offered {
		return nil, signin.ErrSignInOff
	}

	address, err := signin.AddressOf(in.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to read the address: %w", err)
	}
	if u.blocklist.Disposable(address.Domain()) {
		return nil, fmt.Errorf("%w: %s", signin.ErrAddressDisposable, address.Domain())
	}

	if in.IP == "" {
		return nil, fmt.Errorf("%w: the request carries no source address", attestation.ErrAttestationFailed)
	}
	// Before the address spends its budget, so a caller that proves nothing cannot keep its owner from signing in.
	if err := u.attester.Attest(ctx, in.AttestationToken, in.IP); err != nil {
		return nil, fmt.Errorf("%w: %w", attestation.ErrAttestationFailed, err)
	}

	now := u.clock.Now()
	var account accounts.AccountID
	if in.Intent == accounts.IntentLink {
		session, err := accounts.Caller(ctx, u.sessions, in.CookieHeader, now)
		if err != nil {
			return nil, fmt.Errorf("failed to find the account to link to: %w", err)
		}
		account = session.Account
	}

	if allowed, _ := u.sends.Take(string(address)); !allowed {
		return nil, signin.ErrTooManyCodes
	}

	challenge, err := signin.NewChallenge(address, in.Intent, account, u.secrets, u.codes, now)
	if err != nil {
		return nil, fmt.Errorf("failed to start the challenge: %w", err)
	}
	sealed, err := u.sealer.SealedChallenge(challenge)
	if err != nil {
		return nil, fmt.Errorf("failed to seal the challenge: %w", err)
	}
	if err := u.mailer.Send(ctx, address, signin.CodeLetter(challenge.Code)); err != nil {
		return nil, fmt.Errorf("failed to send the code: %w", err)
	}

	return &Out{SetCookie: challenge.Cookie(sealed, now)}, nil
}
