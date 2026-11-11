package offercaller

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Of is who a request comes from, in the offers' words: the edge reads the click token and the address and hands them
// over, since bonuses reads no request of its own.
func Of(ctx context.Context) bonuses.Caller {
	payer := clicks.PayerOf(ctx)

	return bonuses.Caller{Scope: payer.Scope, Account: bonuses.Account(payer.Account), Linked: payer.Linked}
}
