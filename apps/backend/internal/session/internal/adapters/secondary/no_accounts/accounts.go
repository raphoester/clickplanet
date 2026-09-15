// Package no_accounts is who every caller is when the server runs without accounts: nobody.
package no_accounts

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
)

type Accounts struct{}

var _ domain.Accounts = Accounts{}

func (Accounts) Resolve(context.Context, string, bool) (domain.Resolution, error) {
	return domain.Resolution{}, nil
}
