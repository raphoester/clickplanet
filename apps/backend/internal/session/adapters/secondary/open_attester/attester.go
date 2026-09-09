// Package open_attester mints a session for anyone who asks.
//
// It is what runs with turnstile disabled, which is how a local backend works
// without a widget and a secret. It is not a degraded Turnstile: with it in
// place the session token is still bound and still expires, so the click path
// is exercised exactly as in production, but nothing had to be proved to get
// one. Never the production choice.
package open_attester

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/domain"
)

type Attester struct{}

var _ domain.Attester = (*Attester)(nil)

func New() *Attester {
	return &Attester{}
}

func (*Attester) Attest(context.Context, string, string) error {
	return nil
}
