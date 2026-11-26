package refund_refused_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

type Refunder interface {
	Refund(ctx context.Context, intent payments.IntentID) error
	Cancel(ctx context.Context, id renewals.SubscriptionID) error
}

func New(paid payments.Store, renewed renewals.Store, stripe Refunder) *UseCase {
	return &UseCase{paid: paid, renewed: renewed, stripe: stripe}
}

type UseCase struct {
	paid    payments.Store
	renewed renewals.Store
	stripe  Refunder
}

// Execute refunds in full a payment the flags module refused: it bought no time. Stripe's refund takes nothing back,
// since nothing was granted. The buyer's renewals of its target are cancelled first: the order never buys time, so a
// next charge would be refused too, and a renewal that runs keeps every other player from claiming the flag.
func (u *UseCase) Execute(ctx context.Context, grant string) error {
	id, err := payments.PaymentOfGrant(grant)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	payment, err := u.paid.Payment(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to read the payment refused: %w", err)
	}
	order := payment.TakeSnapshot().Order
	renewing, err := u.renewed.OfTarget(ctx, order.Target)
	if err != nil {
		return fmt.Errorf("failed to read the renewals of the payment refused: %w", err)
	}
	for _, subscription := range renewals.CancellableOf(renewing, order.Buyer) {
		if err := u.stripe.Cancel(ctx, subscription); err != nil {
			return err //nolint:wrapcheck // the gateway names it.
		}
	}
	intent, ok := payment.Refundable()
	if !ok {
		return nil
	}
	return u.stripe.Refund(ctx, intent) //nolint:wrapcheck // the gateway names it.
}
