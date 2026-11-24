package renewals

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

type Store interface {
	// ErrUnknownRenewal for one never kept. Locked to the end of the transaction a change reads it in.
	Renewal(ctx context.Context, id SubscriptionID) (Renewal, error)
	// Every renewal of the buyer, the ended ones too.
	Bought(ctx context.Context, buyer orders.AccountID) ([]Renewal, error)
	Save(ctx context.Context, renewal Renewal) error
}
