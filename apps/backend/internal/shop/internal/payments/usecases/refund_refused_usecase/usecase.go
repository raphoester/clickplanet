package refund_refused_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
)

type Refunder interface {
	Refund(ctx context.Context, intent payments.IntentID) error
}

func New(paid payments.Store, stripe Refunder) *UseCase {
	return &UseCase{paid: paid, stripe: stripe}
}

type UseCase struct {
	paid   payments.Store
	stripe Refunder
}

// Execute refunds in full a payment the flags module refused: it bought no time. Stripe's refund takes nothing back,
// since nothing was granted.
func (u *UseCase) Execute(ctx context.Context, grant string) error {
	id, err := payments.PaymentOfGrant(grant)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	payment, err := u.paid.Payment(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to read the payment refused: %w", err)
	}
	intent, ok := payment.Refundable()
	if !ok {
		return nil
	}
	return u.stripe.Refund(ctx, intent) //nolint:wrapcheck // the gateway names it.
}
