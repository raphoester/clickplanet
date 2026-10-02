package attestation

import (
	"context"
	"errors"
)

var ErrAttestationFailed = errors.New("attestation failed")

type Attester interface {
	Attest(ctx context.Context, token string, ip string) error
}
