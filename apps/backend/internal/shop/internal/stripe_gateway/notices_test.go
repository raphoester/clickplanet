package stripe_gateway_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/stripe_gateway"
)

const (
	account    = "00000000-0000-0000-0000-00000000000a"
	submission = "00000000-0000-0000-0000-000000000001"
)

func read(t *testing.T, payload []byte, signature string) notices.Notice {
	t.Helper()
	notice, err := stripe_gateway.NewReader(stripe_gateway.TestSecret).Read(payload, signature)
	require.NoError(t, err)
	return notice
}

func TestAWeekPaidIsReadAsAPaymentOfItsOrder(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeCheckoutSessionCompleted,
		stripe_gateway.Session("cs_1", "pi_1", stripe_gateway.Metadata(account, submission, "week")), now)

	notice := read(t, payload, signature)

	target, err := orders.ParseTarget(submission, "")
	require.NoError(t, err)
	buyer, err := orders.ParseAccountID(account)
	require.NoError(t, err)
	assert.Equal(t, notices.Notice{
		Event: "evt_1", Kind: notices.Paid, At: now, Payment: payments.Charged{
			ID: "cs_1", Intent: "pi_1", Amount: 400, Currency: "eur", At: now, Order: orders.Snapshot{
				Buyer: buyer, Plan: plans.Week, Target: target,
				AgreedAt: time.Date(2026, 11, 24, 8, 59, 0, 0, time.UTC),
			},
		},
	}, notice)
}

func TestARenewalsInvoiceIsReadAsAPaymentWhoseIntentIsAskedLater(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeInvoicePaid,
		stripe_gateway.Invoice("in_1", "sub_1", stripe_gateway.Metadata(account, submission, "month")), now)

	notice := read(t, payload, signature)

	assert.Equal(t, notices.Paid, notice.Kind)
	assert.True(t, notice.Invoice)
	assert.Equal(t, payments.PaymentID("in_1"), notice.Payment.ID)
	assert.Equal(t, plans.Month, notice.Payment.Order.Plan)
	assert.Empty(t, notice.Payment.Intent)
}

func TestAnInvoiceOfAChangeToTheSubscriptionPaysNoPeriod(t *testing.T) {
	for _, reason := range []string{"subscription_update", "manual", "subscription_threshold"} {
		invoice := stripe_gateway.Invoice("in_1", "sub_1", stripe_gateway.Metadata(account, submission, "month"))
		invoice["billing_reason"], invoice["amount_paid"] = reason, 0
		payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeInvoicePaid, invoice, time.Now())

		assert.Equal(t, notices.Ignored, read(t, payload, signature).Kind, reason)
	}
}

func TestARenewingPlansSessionIsLeftToItsInvoice(t *testing.T) {
	session := stripe_gateway.Session("cs_1", "", stripe_gateway.Metadata(account, submission, "month"))
	session["mode"], session["payment_intent"] = "subscription", nil
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeCheckoutSessionCompleted, session, time.Now())

	assert.Equal(t, notices.Ignored, read(t, payload, signature).Kind)
}

func TestASessionNotPaidYetIsLeftToItsLaterEvent(t *testing.T) {
	session := stripe_gateway.Session("cs_1", "pi_1", stripe_gateway.Metadata(account, submission, "week"))
	session["payment_status"] = "unpaid"
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeCheckoutSessionCompleted, session, time.Now())

	assert.Equal(t, notices.Ignored, read(t, payload, signature).Kind)
}

func TestWhatTheShopNeverSoldIsIgnored(t *testing.T) {
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeCheckoutSessionCompleted,
		stripe_gateway.Session("cs_1", "pi_1", map[string]any{"course": "coaching"}), time.Now())

	notice := read(t, payload, signature)

	assert.Equal(t, notices.Ignored, notice.Kind)
	assert.Contains(t, notice.Why, "not the shop's")
}

func TestAnEventTheShopDoesNotActOnIsIgnored(t *testing.T) {
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeCustomerCreated,
		map[string]any{"id": "cus_1", "object": "customer"}, time.Now())

	assert.Equal(t, notices.Ignored, read(t, payload, signature).Kind)
}

func TestARefundInFullTakesThePaymentBack(t *testing.T) {
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeChargeRefunded,
		stripe_gateway.Charge("pi_1", true), time.Now())

	notice := read(t, payload, signature)

	assert.Equal(t, notices.TakenBack, notice.Kind)
	assert.Equal(t, payments.IntentID("pi_1"), notice.Intent)
}

func TestARefundInPartTakesNothingBack(t *testing.T) {
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeChargeRefunded,
		stripe_gateway.Charge("pi_1", false), time.Now())

	assert.Equal(t, notices.Ignored, read(t, payload, signature).Kind)
}

func TestADisputeLostTakesThePaymentBackAndOneWonDoesNot(t *testing.T) {
	lost, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeChargeDisputeClosed,
		stripe_gateway.Dispute("pi_1", "lost"), time.Now())
	assert.Equal(t, notices.TakenBack, read(t, lost, signature).Kind)

	won, signature := stripe_gateway.Signed("evt_2", stripe.EventTypeChargeDisputeClosed,
		stripe_gateway.Dispute("pi_1", "won"), time.Now())
	assert.Equal(t, notices.Ignored, read(t, won, signature).Kind)
}

func TestASubscriptionIsReadAsItsRenewalsState(t *testing.T) {
	ends := time.Date(2026, 12, 24, 9, 0, 0, 0, time.UTC)
	metadata := stripe_gateway.Metadata(account, submission, "month")
	for name, c := range map[string]struct {
		kind     stripe.EventType
		status   string
		stopping bool
		state    renewals.State
	}{
		"active":            {stripe.EventTypeCustomerSubscriptionUpdated, "active", false, renewals.Renewing},
		"stopped":           {stripe.EventTypeCustomerSubscriptionUpdated, "active", true, renewals.Stopping},
		"retried":           {stripe.EventTypeCustomerSubscriptionUpdated, "past_due", false, renewals.Renewing},
		"never paid":        {stripe.EventTypeCustomerSubscriptionUpdated, "incomplete_expired", false, renewals.Ended},
		"deleted":           {stripe.EventTypeCustomerSubscriptionDeleted, "canceled", false, renewals.Ended},
		"created, not paid": {stripe.EventTypeCustomerSubscriptionCreated, "incomplete", false, renewals.Ended},
	} {
		t.Run(name, func(t *testing.T) {
			payload, signature := stripe_gateway.Signed("evt_1", c.kind,
				stripe_gateway.Subscription("sub_1", c.status, c.stopping, ends, metadata), time.Now())

			notice := read(t, payload, signature)

			require.Equal(t, notices.Renewal, notice.Kind)
			assert.Equal(t, renewals.SubscriptionID("sub_1"), notice.Renewal.ID)
			assert.Equal(t, c.state, notice.Renewal.State)
			assert.Equal(t, ends, notice.Renewal.PeriodEndsAt)
		})
	}
}

func TestANoticeStripeDidNotSignIsRefused(t *testing.T) {
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeCheckoutSessionCompleted,
		stripe_gateway.Session("cs_1", "pi_1", stripe_gateway.Metadata(account, submission, "week")), time.Now())

	_, err := stripe_gateway.NewReader("whsec_another").Read(payload, signature)
	require.ErrorIs(t, err, notices.ErrNotStripes)

	_, err = stripe_gateway.NewReader(stripe_gateway.TestSecret).Read(append(payload, ' '), signature)
	require.ErrorIs(t, err, notices.ErrNotStripes, "a payload changed after its signature")
}

func TestANoticeSignedMoreThanFiveMinutesAgoIsRefused(t *testing.T) {
	payload, signature := stripe_gateway.Signed("evt_1", stripe.EventTypeCheckoutSessionCompleted,
		stripe_gateway.Session("cs_1", "pi_1", stripe_gateway.Metadata(account, submission, "week")),
		time.Now().Add(-6*time.Minute))

	_, err := stripe_gateway.NewReader(stripe_gateway.TestSecret).Read(payload, signature)
	require.ErrorIs(t, err, notices.ErrNotStripes)
}

func TestANoticeOfAnotherAPIVersionIsNotRead(t *testing.T) {
	payload, signature := stripe_gateway.SignedIn("2025-08-27.basil", "evt_1", stripe.EventTypeCheckoutSessionCompleted,
		stripe_gateway.Session("cs_1", "pi_1", stripe_gateway.Metadata(account, submission, "week")), time.Now())

	_, err := stripe_gateway.NewReader(stripe_gateway.TestSecret).Read(payload, signature)
	require.ErrorIs(t, err, notices.ErrInvalidNote)
	require.NotErrorIs(t, err, notices.ErrNotStripes)
}
