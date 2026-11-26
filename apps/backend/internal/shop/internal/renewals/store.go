package renewals

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

type Store interface {
	// ErrUnknownRenewal for one never kept. Locked to the end of the transaction a change reads it in.
	Renewal(ctx context.Context, id SubscriptionID) (Renewal, error)
	// Every renewal for the target, the ended ones too.
	OfTarget(ctx context.Context, target orders.Target) ([]Renewal, error)
	Save(ctx context.Context, renewal Renewal) error
}
