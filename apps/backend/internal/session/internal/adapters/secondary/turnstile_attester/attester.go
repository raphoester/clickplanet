package turnstile_attester

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/turnstile"
	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
)

type Verifier interface {
	Verify(ctx context.Context, token string, remoteIP string) error
}

type Attester struct {
	verifier Verifier
}

var _ domain.Attester = (*Attester)(nil)

func New(config turnstile.Config) (*Attester, error) {
	client, err := turnstile.New(config)
	if err != nil {
		return nil, err
	}

	return &Attester{verifier: client}, nil
}

func (a *Attester) Attest(ctx context.Context, token string, ip string) error {
	return a.verifier.Verify(ctx, token, ip)
}
