//go:build testing

package stripe_gateway

import (
	"encoding/json"
	"time"

	"github.com/stripe/stripe-go/v87"
	"github.com/stripe/stripe-go/v87/webhook"
)

// TestSecret is the webhook endpoint's secret the tests sign with.
const TestSecret = "whsec_test_secret" //nolint:gosec // a test's secret: it signs nothing real.

// Signed is a Stripe event of a kind about an object, as Stripe signs it at a time, and the Stripe-Signature header.
func Signed(id string, kind stripe.EventType, object map[string]any, at time.Time) ([]byte, string) {
	return SignedIn(stripe.APIVersion, id, kind, object, at)
}

// SignedIn is an event of an API version.
func SignedIn(version, id string, kind stripe.EventType, object map[string]any, at time.Time) ([]byte, string) {
	payload, err := json.Marshal(map[string]any{
		"id": id, "object": "event", "api_version": version, "created": at.Unix(), "type": string(kind),
		"livemode": false, "pending_webhooks": 1, "data": map[string]any{"object": object},
	})
	if err != nil {
		panic(err)
	}
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{
		Payload: payload, Secret: TestSecret, Timestamp: at,
	})
	return signed.Payload, signed.Header
}

// Metadata is what the shop gives Stripe of an order: a week for a submission, by an account.
func Metadata(account, submission, plan string) map[string]any {
	return map[string]any{
		"account_id": account, "submission_id": submission, "plan": plan, "starts_now_agreed_at": "2026-11-24T08:59:00Z",
	}
}

// Session is a week's Checkout Session, paid.
func Session(id, intent string, metadata map[string]any) map[string]any {
	return map[string]any{
		"id": id, "object": "checkout.session", "mode": "payment", "payment_status": "paid", "payment_intent": intent,
		"amount_total": 400, "currency": "eur", "metadata": metadata, "client_reference_id": metadata["account_id"],
	}
}

// Invoice is a month's or a year's invoice, paid, of a subscription.
func Invoice(id, subscription string, metadata map[string]any) map[string]any {
	return map[string]any{
		"id": id, "object": "invoice", "amount_paid": 1000, "currency": "eur", "billing_reason": "subscription_cycle",
		"parent": map[string]any{
			"type":                 "subscription_details",
			"subscription_details": map[string]any{"subscription": subscription, "metadata": metadata},
		},
	}
}

// Subscription is a month's subscription, with the end of its period.
func Subscription(id, status string, cancelAtPeriodEnd bool, periodEnd time.Time, metadata map[string]any) map[string]any {
	return map[string]any{
		"id": id, "object": "subscription", "status": status, "cancel_at_period_end": cancelAtPeriodEnd,
		"metadata": metadata, "items": map[string]any{
			"object": "list", "data": []any{map[string]any{
				"id": "si_1", "object": "subscription_item", "current_period_end": periodEnd.Unix(),
			}},
		},
	}
}

// Charge is a charge of an intent, refunded in full or in part.
func Charge(intent string, refunded bool) map[string]any {
	return map[string]any{
		"id": "ch_1", "object": "charge", "payment_intent": intent, "refunded": refunded, "amount": 400,
		"amount_refunded": 400,
	}
}

// Dispute is a dispute of an intent, closed with a status.
func Dispute(intent, status string) map[string]any {
	return map[string]any{"id": "dp_1", "object": "dispute", "payment_intent": intent, "charge": "ch_1", "status": status}
}
