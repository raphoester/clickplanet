// Package create_anonymous_session_usecase mints a click token with no account, for clients that predate accounts.
//
// Deprecated: create_session_usecase replaces it, and it goes with session.v1.
package create_anonymous_session_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Minter interface {
	Mint(ip string, account uuid.UUID, now time.Time) (*cpsession.Token, error)
}

type In struct {
	AttestationToken string
	IP               string
}

type UseCase struct {
	attester attestation.Attester
	minter   Minter
	clock    cptime.Clock
}

func New(attester attestation.Attester, minter Minter, clock cptime.Clock) *UseCase {
	return &UseCase{attester: attester, minter: minter, clock: clock}
}

// Execute answers attestation.ErrAttestationFailed for a caller that proved nothing.
func (u *UseCase) Execute(ctx context.Context, in In) (*cpsession.Token, error) {
	if in.IP == "" {
		return nil, fmt.Errorf("%w: the request carries no source address", attestation.ErrAttestationFailed)
	}
	if err := u.attester.Attest(ctx, in.AttestationToken, in.IP); err != nil {
		return nil, fmt.Errorf("%w: %w", attestation.ErrAttestationFailed, err)
	}

	token, err := u.minter.Mint(in.IP, uuid.Nil, u.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to mint the click token: %w", err)
	}
	return token, nil
}
