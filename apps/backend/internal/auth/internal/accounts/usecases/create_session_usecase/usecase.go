package create_session_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Resumer interface {
	Resume(ctx context.Context, cookieHeader string, now time.Time) (accounts.Entry, error)
}

type Guests interface {
	Start(ctx context.Context, now time.Time) (accounts.Entry, error)
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
	resumer  Resumer
	guests   Guests
	minter   Minter
	clock    cptime.Clock
}

func New(attester attestation.Attester, resumer Resumer, guests Guests, minter Minter, clock cptime.Clock) *UseCase {
	return &UseCase{attester: attester, resumer: resumer, guests: guests, minter: minter, clock: clock}
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

	entry, err := u.resumer.Resume(ctx, in.CookieHeader, now)
	if accounts.Ended(err) {
		entry, err = u.guests.Start(ctx, now)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller's account: %w", err)
	}

	token, err := u.minter.Mint(in.IP,
		cpsession.Holder{Account: entry.Account(), Linked: entry.Linked(), Attested: true}, now)
	if err != nil {
		return nil, fmt.Errorf("failed to mint the click token: %w", err)
	}

	return &Out{Token: token, Account: entry.Account(), Linked: entry.Linked(), SetCookie: entry.SetCookie()}, nil
}
