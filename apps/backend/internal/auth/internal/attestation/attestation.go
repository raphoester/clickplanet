// Package attestation is what a caller has to prove before a click token is minted for it: that it is not a script.
package attestation

import (
	"context"
	"errors"
)

var ErrAttestationFailed = errors.New("attestation failed")

// Attester is Turnstile in production and open_attester locally; the use cases do not care which.
type Attester interface {
	Attest(ctx context.Context, token string, ip string) error
}
