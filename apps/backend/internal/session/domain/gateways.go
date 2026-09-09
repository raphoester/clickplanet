package domain

import "context"

// Attester is whatever the caller had to satisfy before it is worth minting a
// session for them. Turnstile is one implementation; the open one is another,
// and the domain does not care which it was handed.
type Attester interface {
	Attest(ctx context.Context, token string, ip string) error
}
