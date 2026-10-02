package open_attester

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/attestation"
)

type Attester struct{}

var _ attestation.Attester = (*Attester)(nil)

func New() *Attester {
	return &Attester{}
}

func (*Attester) Attest(context.Context, string, string) error {
	return nil
}
