package stripe_gateway

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

// Off is Stripe with no key in the config: every call is plans.ErrOff.
type Off struct{}

func (Off) Checkout(context.Context, orders.Order, plans.Price) (string, error) {
	return "", plans.ErrOff
}

func (Off) StopAtPeriodEnd(context.Context, renewals.SubscriptionID) error {
	return plans.ErrOff
}

func (Off) IntentOf(context.Context, payments.PaymentID) (payments.IntentID, error) {
	return "", plans.ErrOff
}
