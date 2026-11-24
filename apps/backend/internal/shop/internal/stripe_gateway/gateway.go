package stripe_gateway

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/stripe/stripe-go/v87"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

var ErrNoIntent = errors.New("no paid intent of the invoice at Stripe")

// What the page reads in the address it comes back to.
const (
	paidQuery      = "paid"
	cancelledQuery = "cancelled"
)

// New talks to Stripe with the shop's key, and sends the buyer back to the game's page after Stripe's.
func New(client *stripe.Client, returnURL *url.URL) Gateway {
	return Gateway{client: client, returnURL: returnURL}
}

// Gateway is Stripe's API, in the shop's words. It decides nothing.
type Gateway struct {
	client    *stripe.Client
	returnURL *url.URL
}

// Checkout makes a Stripe Checkout Session under Managed Payments, Stripe being the seller, and answers its page.
func (g Gateway) Checkout(ctx context.Context, order orders.Order, price plans.Price) (string, error) {
	kept := order.TakeSnapshot()
	metadata := order.Metadata()
	params := &stripe.CheckoutSessionCreateParams{
		LineItems:         []*stripe.CheckoutSessionCreateLineItemParams{{Price: stripe.String(price.ID.String()), Quantity: stripe.Int64(1)}},
		ManagedPayments:   &stripe.CheckoutSessionCreateManagedPaymentsParams{Enabled: stripe.Bool(true)},
		ClientReferenceID: stripe.String(kept.Buyer.String()),
		Metadata:          metadata,
		SuccessURL:        stripe.String(g.back(paidQuery)),
		CancelURL:         stripe.String(g.back(cancelledQuery)),
	}
	if kept.Plan.Renews() {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModeSubscription))
		params.SubscriptionData = &stripe.CheckoutSessionCreateSubscriptionDataParams{Metadata: metadata}
	} else {
		params.Mode = stripe.String(string(stripe.CheckoutSessionModePayment))
		params.PaymentIntentData = &stripe.CheckoutSessionCreatePaymentIntentDataParams{Metadata: metadata}
	}
	session, err := g.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to make a Stripe Checkout Session: %w", err)
	}
	return session.URL, nil
}

func (g Gateway) back(outcome string) string {
	back := *g.returnURL
	query := back.Query()
	query.Set("shop", outcome)
	back.RawQuery = query.Encode()
	return back.String()
}

// StopAtPeriodEnd stops a subscription at the end of the period paid: the buyer keeps what it paid for.
func (g Gateway) StopAtPeriodEnd(ctx context.Context, id renewals.SubscriptionID) error {
	if _, err := g.client.V1Subscriptions.Update(ctx, id.String(), &stripe.SubscriptionUpdateParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}); err != nil {
		return fmt.Errorf("failed to stop the subscription %s at its period's end: %w", id, err)
	}
	return nil
}

// IntentOf is the PaymentIntent that paid an invoice: since API version 2025-03-31, an invoice no longer names it, and
// a refund names only the intent.
func (g Gateway) IntentOf(ctx context.Context, invoice payments.PaymentID) (payments.IntentID, error) {
	list := g.client.V1InvoicePayments.List(ctx, &stripe.InvoicePaymentListParams{
		Invoice: stripe.String(invoice.String()), Status: stripe.String("paid"),
	})
	for paid, err := range list.All(ctx) {
		if err != nil {
			return "", fmt.Errorf("failed to list the payments of the invoice %s: %w", invoice, err)
		}
		if paid.Payment != nil && paid.Payment.PaymentIntent != nil {
			return payments.ParseIntentID(paid.Payment.PaymentIntent.ID) //nolint:wrapcheck // the parse names it.
		}
	}
	return "", fmt.Errorf("%w: %s", ErrNoIntent, invoice)
}
