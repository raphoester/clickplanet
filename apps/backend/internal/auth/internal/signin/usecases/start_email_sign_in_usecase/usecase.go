package start_email_sign_in_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

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
	attester   attestation.Attester
	sessions   accounts.SessionFinder
	challenges *signin.Challenges
	post       *signin.Post
	clock      cptime.Clock
}

func New(
	attester attestation.Attester, sessions accounts.SessionFinder, challenges *signin.Challenges, post *signin.Post, clock cptime.Clock,
) *UseCase {
	return &UseCase{attester: attester, sessions: sessions, challenges: challenges, post: post, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	if u.challenges.Off() {
		return nil, signin.ErrSignInOff
	}
	address, err := signin.AddressOf(in.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to read the address: %w", err)
	}
	// Before the post spends the address's budget, or a script could lock its owner out.
	if err := u.attester.Attest(ctx, in.AttestationToken, in.IP); err != nil {
		return nil, fmt.Errorf("%w: %w", attestation.ErrAttestationFailed, err)
	}

	now := u.clock.Now()
	account, err := signin.LinkTarget(ctx, u.sessions, in.Intent, in.CookieHeader, now)
	if err != nil {
		return nil, fmt.Errorf("failed to start the sign-in: %w", err)
	}
	challenge, setCookie, err := u.challenges.Issued(address, in.Intent, account, now)
	if err != nil {
		return nil, fmt.Errorf("failed to issue the challenge: %w", err)
	}
	if err := u.post.Send(ctx, address, challenge.Letter()); err != nil {
		return nil, fmt.Errorf("failed to post the code: %w", err)
	}
	return &Out{SetCookie: setCookie}, nil
}
