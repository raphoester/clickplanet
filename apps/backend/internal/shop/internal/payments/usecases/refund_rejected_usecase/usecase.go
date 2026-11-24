package refund_rejected_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
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

// Execute cancels the renewals of a rejected submission, then refunds every payment for it: Stripe's refunds take its
// time back as any refund does. Told again, it refunds and cancels nothing twice.
func (u *UseCase) Execute(ctx context.Context, submission string) error {
	target, err := orders.ParseTarget(submission, "")
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	renewing, err := u.renewed.OfTarget(ctx, target)
	if err != nil {
		return fmt.Errorf("failed to read the renewals of %s: %w", submission, err)
	}
	for _, renewal := range renewing {
		if id, ok := renewal.Cancellable(); ok {
			if err := u.stripe.Cancel(ctx, id); err != nil {
				return err //nolint:wrapcheck // the gateway names it.
			}
		}
	}
	paid, err := u.paid.OfTarget(ctx, target)
	if err != nil {
		return fmt.Errorf("failed to read the payments for %s: %w", submission, err)
	}
	for _, payment := range paid {
		if intent, ok := payment.Refundable(); ok {
			if err := u.stripe.Refund(ctx, intent); err != nil {
				return err //nolint:wrapcheck // the gateway names it.
			}
		}
	}
	return nil
}
