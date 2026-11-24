package refund_rejected_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/checkouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

type Refunder interface {
	Expire(ctx context.Context, id checkouts.SessionID) error
	Refund(ctx context.Context, intent payments.IntentID) error
	Cancel(ctx context.Context, id renewals.SubscriptionID) error
}

func New(
	clock cptime.Clock, opened checkouts.Store, paid payments.Store, renewed renewals.Store, stripe Refunder,
) *UseCase {
	return &UseCase{clock: clock, opened: opened, paid: paid, renewed: renewed, stripe: stripe}
}

type UseCase struct {
	clock   cptime.Clock
	opened  checkouts.Store
	paid    payments.Store
	renewed renewals.Store
	stripe  Refunder
}

// Execute closes the pages still open for a rejected submission, cancels its renewals, then refunds every payment for
// it: Stripe's refunds take its time back as any refund does. Told again, it refunds and cancels nothing twice.
func (u *UseCase) Execute(ctx context.Context, submission string) error {
	target, err := orders.ParseTarget(submission, "")
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	pages, err := u.opened.OfTarget(ctx, target)
	if err != nil {
		return fmt.Errorf("failed to read the pages opened for %s: %w", submission, err)
	}
	for _, page := range pages {
		if id, ok := page.Expirable(u.clock.Now()); ok {
			if err := u.stripe.Expire(ctx, id); err != nil {
				return err //nolint:wrapcheck // the gateway names it.
			}
		}
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
