// Package session_service holds the one rule this context has: a session is
// minted only for a caller that passed attestation, and it is bound to the
// address that passed it.
package session_service

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Minter interface {
	Mint(ip string, now time.Time) (cpsession.Token, error)
}

type IService interface {
	Create(ctx context.Context, attestationToken string, ip string) (cpsession.Token, error)
}

type Service struct {
	attester domain.Attester
	minter   Minter
	clock    cptime.Clock
}

var _ IService = (*Service)(nil)

func New(attester domain.Attester, minter Minter, clock cptime.Clock) *Service {
	if clock == nil {
		clock = cptime.SystemClock{}
	}

	return &Service{attester: attester, minter: minter, clock: clock}
}

func (s *Service) Create(ctx context.Context, attestationToken string, ip string) (cpsession.Token, error) {
	// A session is an address that proved something. Minting one against no
	// address at all would produce a token every caller could use, since
	// verification would bind to the same empty string.
	if ip == "" {
		return cpsession.Token{}, fmt.Errorf("%w: the request carries no source address", domain.ErrAttestationFailed)
	}

	if err := s.attester.Attest(ctx, attestationToken, ip); err != nil {
		return cpsession.Token{}, fmt.Errorf("%w: %w", domain.ErrAttestationFailed, err)
	}

	token, err := s.minter.Mint(ip, s.clock.Now())
	if err != nil {
		return cpsession.Token{}, fmt.Errorf("failed to mint a session token: %w", err)
	}

	return token, nil
}
