package domain

import (
	"context"

	"github.com/google/uuid"
)

// Attester is whatever the caller had to satisfy before it is worth minting a
// session for them. Turnstile is one implementation; the open one is another,
// and the domain does not care which it was handed.
type Attester interface {
	Attest(ctx context.Context, token string, ip string) error
}

type Accounts interface {
	Resolve(ctx context.Context, cookieHeader string, create bool) (Resolution, error)
}

// Resolution is an account (uuid.Nil for none) and the Set-Cookie to pass back as is (empty for none).
type Resolution struct {
	Account   uuid.UUID
	SetCookie string
}
