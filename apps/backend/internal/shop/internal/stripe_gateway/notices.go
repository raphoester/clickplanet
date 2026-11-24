package stripe_gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

// Stripe's own default: a notice signed longer ago is refused, so a captured one cannot be sent again later.
const tolerance = 5 * time.Minute

// NewReader checks Stripe's signature with the webhook endpoint's secret (whsec_), and reads each event in the
// shop's words.
func NewReader(secret string) Reader {
	return Reader{secret: secret}
}

type Reader struct {
	secret string
}

var _ notices.Reader = Reader{}

func (r Reader) Read(payload []byte, signature string) (notices.Notice, error) {
	event, err := webhook.ConstructEventWithOptions(payload, signature, r.secret,
		webhook.ConstructEventOptions{Tolerance: tolerance})
	switch {
	case errors.Is(err, webhook.ErrInvalidHeader), errors.Is(err, webhook.ErrNoValidSignature),
		errors.Is(err, webhook.ErrNotSigned), errors.Is(err, webhook.ErrTooOld), errors.Is(err, webhook.ErrEmptySecret):
		return notices.Notice{}, fmt.Errorf("%w: %w", notices.ErrNotStripes, err)
	// Signed, but not read: another API version than the endpoint's, or no event.
	case err != nil:
		return notices.Notice{}, fmt.Errorf("%w: %w", notices.ErrInvalidNote, err)
	}
	id, err := notices.ParseEventID(event.ID)
	if err != nil {
		return notices.Notice{}, err //nolint:wrapcheck // the parse names it.
	}
	notice := notices.Notice{Event: id, At: time.Unix(event.Created, 0).UTC()}
	read, ok := readOf(event.Type)
	if !ok {
		return ignored(notice, "not an event the shop acts on: "+string(event.Type)), nil
	}
	if err := read(&notice, event.Data.Raw); err != nil {
		if errors.Is(err, orders.ErrNotOurs) {
			return ignored(notice, "not the shop's: "+string(event.Type)), nil
		}
		return notices.Notice{}, fmt.Errorf("%w: %s %s: %w", notices.ErrInvalidNote, event.Type, event.ID, err)
	}
	return notice, nil
}

func ignored(notice notices.Notice, why string) notices.Notice {
	notice.Kind, notice.Why = notices.Ignored, why
	return notice
}

type read func(notice *notices.Notice, raw json.RawMessage) error

// Stripe has hundreds of kinds of event: the endpoint may send more than these.
func readOf(kind stripe.EventType) (read, bool) {
	switch kind {
	case stripe.EventTypeCheckoutSessionCompleted, stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		return session, true
	case stripe.EventTypeInvoicePaid:
		return invoice, true
	case stripe.EventTypeChargeRefunded:
		return refund, true
	case stripe.EventTypeChargeDisputeClosed:
		return dispute, true
	case stripe.EventTypeCustomerSubscriptionCreated, stripe.EventTypeCustomerSubscriptionUpdated:
		return subscription(false), true
	case stripe.EventTypeCustomerSubscriptionDeleted:
		return subscription(true), true
	default:
		return nil, false
	}
}

// A week paid. A month's or a year's session is paid by its first invoice, told as invoice.paid; a session paid later
// (a bank debit) is told again as async_payment_succeeded.
func session(notice *notices.Notice, raw json.RawMessage) error {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(raw, &session); err != nil {
		return fmt.Errorf("failed to read the session: %w", err)
	}
	order, err := orders.FromMetadata(session.Metadata)
	if err != nil {
		return err //nolint:wrapcheck // the order names it.
	}
	if session.Mode != stripe.CheckoutSessionModePayment {
		*notice = ignored(*notice, "a renewing plan is paid by its invoice")
		return nil
	}
	if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
		*notice = ignored(*notice, "not paid yet: "+string(session.PaymentStatus))
		return nil
	}
	id, err := payments.ParsePaymentID(session.ID)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	if session.PaymentIntent == nil {
		return fmt.Errorf("the session %s names no intent", session.ID)
	}
	intent, err := payments.ParseIntentID(session.PaymentIntent.ID)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	notice.Kind, notice.Payment = notices.Paid, payments.Charged{
		ID: id, Order: order, Intent: intent, Amount: session.AmountTotal, Currency: string(session.Currency),
		At: notice.At,
	}
	return nil
}

// A month or a year paid: its first invoice, and each renewal.
func invoice(notice *notices.Notice, raw json.RawMessage) error {
	var invoice stripe.Invoice
	if err := json.Unmarshal(raw, &invoice); err != nil {
		return fmt.Errorf("failed to read the invoice: %w", err)
	}
	if invoice.Parent == nil || invoice.Parent.Type != stripe.InvoiceParentTypeSubscriptionDetails ||
		invoice.Parent.SubscriptionDetails == nil {
		return orders.ErrNotOurs
	}
	order, err := orders.FromMetadata(invoice.Parent.SubscriptionDetails.Metadata)
	if err != nil {
		return err //nolint:wrapcheck // the order names it.
	}
	id, err := payments.ParsePaymentID(invoice.ID)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	notice.Kind, notice.Invoice, notice.Payment = notices.Paid, true, payments.Charged{
		ID: id, Order: order, Amount: invoice.AmountPaid, Currency: string(invoice.Currency), At: notice.At,
	}
	return nil
}

// A refund in full takes the payment's time back. A refund in part takes nothing: the operator refunds the unused time
// of a flag closed already.
func refund(notice *notices.Notice, raw json.RawMessage) error {
	var charge stripe.Charge
	if err := json.Unmarshal(raw, &charge); err != nil {
		return fmt.Errorf("failed to read the charge: %w", err)
	}
	if !charge.Refunded {
		*notice = ignored(*notice, "refunded in part")
		return nil
	}
	return takenBack(notice, charge.PaymentIntent)
}

// A dispute lost takes the payment's time back, as a refund does.
func dispute(notice *notices.Notice, raw json.RawMessage) error {
	var dispute stripe.Dispute
	if err := json.Unmarshal(raw, &dispute); err != nil {
		return fmt.Errorf("failed to read the dispute: %w", err)
	}
	if dispute.Status != stripe.DisputeStatusLost {
		*notice = ignored(*notice, "a dispute closed "+string(dispute.Status))
		return nil
	}
	return takenBack(notice, dispute.PaymentIntent)
}

// Refunds and disputes of the account's other sales are no payment of the shop's: the pay step finds none.
func takenBack(notice *notices.Notice, intent *stripe.PaymentIntent) error {
	if intent == nil {
		return orders.ErrNotOurs
	}
	id, err := payments.ParseIntentID(intent.ID)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	notice.Kind, notice.Intent = notices.TakenBack, id
	return nil
}

func subscription(deleted bool) read {
	return func(notice *notices.Notice, raw json.RawMessage) error {
		var subscription stripe.Subscription
		if err := json.Unmarshal(raw, &subscription); err != nil {
			return fmt.Errorf("failed to read the subscription: %w", err)
		}
		order, err := orders.FromMetadata(subscription.Metadata)
		if err != nil {
			return err //nolint:wrapcheck // the order names it.
		}
		id, err := renewals.ParseSubscriptionID(subscription.ID)
		if err != nil {
			return err //nolint:wrapcheck // the parse names it.
		}
		var periodEnd int64
		if subscription.Items != nil {
			for _, item := range subscription.Items.Data {
				periodEnd = max(periodEnd, item.CurrentPeriodEnd)
			}
		}
		notice.Kind, notice.Renewal = notices.Renewal, renewals.Told{
			ID: id, Order: order, State: stateOf(subscription, deleted), PeriodEndsAt: time.Unix(periodEnd, 0).UTC(),
			At: notice.At,
		}
		return nil
	}
}

func stateOf(subscription stripe.Subscription, deleted bool) renewals.State {
	if deleted {
		return renewals.Ended
	}
	switch subscription.Status {
	case stripe.SubscriptionStatusActive, stripe.SubscriptionStatusTrialing, stripe.SubscriptionStatusPastDue:
		if subscription.CancelAtPeriodEnd || subscription.CancelAt != 0 {
			return renewals.Stopping
		}
		return renewals.Renewing
	// Never paid, given up after the retries, or paused: nothing renews.
	case stripe.SubscriptionStatusCanceled, stripe.SubscriptionStatusIncomplete,
		stripe.SubscriptionStatusIncompleteExpired, stripe.SubscriptionStatusUnpaid, stripe.SubscriptionStatusPaused:
		return renewals.Ended
	}
	return renewals.Ended
}
