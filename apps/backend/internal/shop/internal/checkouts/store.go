package checkouts

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

type Store interface {
	Save(ctx context.Context, checkout Checkout) error
	// Every page opened for the target, the first first.
	OfTarget(ctx context.Context, target orders.Target) ([]Checkout, error)
}
