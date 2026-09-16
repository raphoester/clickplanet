package turnstile_attester

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation/turnstile"
)

type Verifier interface {
	Verify(ctx context.Context, token string, remoteIP string) error
}

type Attester struct {
	verifier Verifier
}

var _ attestation.Attester = (*Attester)(nil)

func New(config turnstile.Config) (*Attester, error) {
	client, err := turnstile.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build the turnstile client: %w", err)
	}

	return &Attester{verifier: client}, nil
}

func (a *Attester) Attest(ctx context.Context, token string, ip string) error {
	if err := a.verifier.Verify(ctx, token, ip); err != nil {
		return fmt.Errorf("turnstile refused the token: %w", err)
	}
	return nil
}
