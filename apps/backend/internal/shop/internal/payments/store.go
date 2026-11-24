package payments

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
)

type Store interface {
	// ErrUnknownPayment for one never kept.
	Payment(ctx context.Context, id PaymentID) (Payment, error)
	// The payment of the intent a refund or a dispute names; ErrUnknownPayment for one the shop never took.
	OfIntent(ctx context.Context, intent IntentID) (Payment, error)
	// Every payment for the target, the ones taken back too, the first paid first.
	OfTarget(ctx context.Context, target orders.Target) ([]Payment, error)
	Save(ctx context.Context, payment Payment) error
}
