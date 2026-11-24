package payments

import "context"

type Store interface {
	// ErrUnknownPayment for one never kept.
	Payment(ctx context.Context, id PaymentID) (Payment, error)
	// The payment of the intent a refund or a dispute names; ErrUnknownPayment for one the shop never took.
	OfIntent(ctx context.Context, intent IntentID) (Payment, error)
	Save(ctx context.Context, payment Payment) error
}
