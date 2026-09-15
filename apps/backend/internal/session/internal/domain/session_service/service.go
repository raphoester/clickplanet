// Package session_service holds the one rule this context has: a session is
// minted only for a caller that passed attestation, and it is bound to the
// address that passed it.
package session_service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Minter interface {
	Mint(ip string, account uuid.UUID, now time.Time) (cpsession.Token, error)
}

type IService interface {
	Create(ctx context.Context, request Request) (Minted, error)
}

type Request struct {
	AttestationToken string
	IP               string
	CookieHeader     string
	CreateAccount    bool
}

type Minted struct {
	Token     cpsession.Token
	SetCookie string
}

type Service struct {
	attester domain.Attester
	accounts domain.Accounts
	minter   Minter
	clock    cptime.Clock
}

var _ IService = (*Service)(nil)

func New(attester domain.Attester, accounts domain.Accounts, minter Minter, clock cptime.Clock) *Service {
	return &Service{attester: attester, accounts: accounts, minter: minter, clock: clock}
}

func (s *Service) Create(ctx context.Context, request Request) (Minted, error) {
	// A session is an address that proved something. Minting one against no
	// address at all would produce a token every caller could use, since
	// verification would bind to the same empty string.
	if request.IP == "" {
		return Minted{}, fmt.Errorf("%w: the request carries no source address", domain.ErrAttestationFailed)
	}

	if err := s.attester.Attest(ctx, request.AttestationToken, request.IP); err != nil {
		return Minted{}, fmt.Errorf("%w: %w", domain.ErrAttestationFailed, err)
	}

	// After attestation, so a caller that proved nothing never creates an account.
	resolution, err := s.resolve(ctx, request)
	if err != nil {
		return Minted{}, err
	}

	token, err := s.minter.Mint(request.IP, resolution.Account, s.clock.Now())
	if err != nil {
		return Minted{}, fmt.Errorf("failed to mint a session token: %w", err)
	}

	return Minted{Token: token, SetCookie: resolution.SetCookie}, nil
}

func (s *Service) resolve(ctx context.Context, request Request) (domain.Resolution, error) {
	if request.CookieHeader == "" && !request.CreateAccount {
		return domain.Resolution{}, nil
	}

	resolution, err := s.accounts.Resolve(ctx, request.CookieHeader, request.CreateAccount)
	if err != nil {
		return domain.Resolution{}, fmt.Errorf("failed to resolve the caller's account: %w", err)
	}

	return resolution, nil
}
