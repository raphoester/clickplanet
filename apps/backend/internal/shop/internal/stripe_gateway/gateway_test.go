package stripe_gateway_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/stripe_gateway"
)

// fakeStripe answers what the gateway asks, and keeps each request: never the real Stripe.
type fakeStripe struct {
	mu       sync.Mutex
	requests []request
	answer   string
	status   int
}

type request struct {
	method, path string
	form         url.Values
	query        url.Values
	header       http.Header
}

func (f *fakeStripe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(body))
	f.mu.Lock()
	f.requests = append(f.requests, request{
		method: r.Method, path: r.URL.Path, form: form, query: r.URL.Query(), header: r.Header.Clone(),
	})
	answer, status := f.answer, f.status
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if status != 0 {
		w.WriteHeader(status)
	}
	_, _ = io.WriteString(w, answer)
}

func (f *fakeStripe) last() request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[len(f.requests)-1]
}

func gateway(t *testing.T, answer string) (stripe_gateway.Gateway, *fakeStripe) {
	t.Helper()
	fake := &fakeStripe{answer: answer}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	client := stripe.NewClient("sk_test_fake", stripe.WithBackends(stripe.NewBackendsWithConfig(&stripe.BackendConfig{
		URL: stripe.String(server.URL), HTTPClient: server.Client(), MaxNetworkRetries: stripe.Int64(0),
	})))
	back, err := url.Parse("https://clickplanet.lol/play")
	require.NoError(t, err)
	return stripe_gateway.New(client, back), fake
}

func order(t *testing.T, plan plans.Plan) orders.Order {
	t.Helper()
	buyer, err := orders.ParseAccountID(account)
	require.NoError(t, err)
	target, err := orders.ParseTarget(submission, "")
	require.NoError(t, err)
	placed, err := orders.Place(buyer, plan, target, true, time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	return placed
}

func TestAWeekIsSoldOnStripesPageUnderManagedPayments(t *testing.T) {
	gate, fake := gateway(t, `{"id":"cs_1","object":"checkout.session","url":"https://checkout.stripe.com/c/pay/cs_1"}`)

	page, err := gate.Checkout(t.Context(), order(t, plans.Week),
		plans.Price{Plan: plans.Week, ID: "price_week", Amount: 400, Currency: "eur"})
	require.NoError(t, err)

	assert.Equal(t, "https://checkout.stripe.com/c/pay/cs_1", page)
	sent := fake.last()
	assert.Equal(t, http.MethodPost, sent.method)
	assert.Equal(t, "/v1/checkout/sessions", sent.path)
	for key, value := range map[string]string{
		"mode":                                      "payment",
		"managed_payments[enabled]":                 "true",
		"line_items[0][price]":                      "price_week",
		"line_items[0][quantity]":                   "1",
		"client_reference_id":                       account,
		"metadata[account_id]":                      account,
		"metadata[submission_id]":                   submission,
		"metadata[plan]":                            "week",
		"metadata[starts_now_agreed_at]":            "2026-11-24T09:00:00Z",
		"payment_intent_data[metadata][account_id]": account,
		"success_url":                               "https://clickplanet.lol/play?shop=paid",
		"cancel_url":                                "https://clickplanet.lol/play?shop=cancelled",
	} {
		assert.Equal(t, value, sent.form.Get(key), key)
	}
	assert.False(t, sent.form.Has("subscription_data[metadata][plan]"))
}

func TestAMonthIsSoldAsASubscriptionThatCarriesItsOrder(t *testing.T) {
	gate, fake := gateway(t, `{"id":"cs_1","object":"checkout.session","url":"https://checkout.stripe.com/c/pay/cs_1"}`)

	_, err := gate.Checkout(t.Context(), order(t, plans.Month),
		plans.Price{Plan: plans.Month, ID: "price_month", Amount: 1000, Currency: "eur"})
	require.NoError(t, err)

	sent := fake.last()
	assert.Equal(t, "subscription", sent.form.Get("mode"))
	assert.Equal(t, "true", sent.form.Get("managed_payments[enabled]"))
	assert.Equal(t, "month", sent.form.Get("subscription_data[metadata][plan]"))
	assert.Equal(t, submission, sent.form.Get("subscription_data[metadata][submission_id]"))
	assert.False(t, sent.form.Has("payment_intent_data[metadata][plan]"))
}

func TestARenewalStopsAtTheEndOfItsPeriod(t *testing.T) {
	gate, fake := gateway(t, `{"id":"sub_1","object":"subscription","cancel_at_period_end":true}`)

	require.NoError(t, gate.StopAtPeriodEnd(t.Context(), "sub_1"))

	sent := fake.last()
	assert.Equal(t, "/v1/subscriptions/sub_1", sent.path)
	assert.Equal(t, "true", sent.form.Get("cancel_at_period_end"))
}

func TestAnInvoicesIntentIsReadFromItsPayments(t *testing.T) {
	gate, fake := gateway(t, `{"object":"list","has_more":false,"url":"/v1/invoice_payments","data":[
		{"id":"inpay_1","object":"invoice_payment","invoice":"in_1","status":"paid",
		 "payment":{"type":"payment_intent","payment_intent":"pi_9"}}]}`)

	intent, err := gate.IntentOf(t.Context(), "in_1")
	require.NoError(t, err)

	assert.Equal(t, payments.IntentID("pi_9"), intent)
	sent := fake.last()
	assert.Equal(t, "/v1/invoice_payments", sent.path)
	assert.Equal(t, "in_1", sent.query.Get("invoice"))
	assert.Equal(t, "paid", sent.query.Get("status"))
}

func TestAnInvoiceWithNoPaidIntentIsAnError(t *testing.T) {
	gate, _ := gateway(t, `{"object":"list","has_more":false,"url":"/v1/invoice_payments","data":[]}`)

	_, err := gate.IntentOf(t.Context(), "in_1")

	require.ErrorIs(t, err, stripe_gateway.ErrNoIntent)
}

func TestARefundGivesAPaymentsMoneyBackOnce(t *testing.T) {
	gate, fake := gateway(t, `{"id":"re_1","object":"refund","payment_intent":"pi_1","status":"succeeded"}`)

	require.NoError(t, gate.Refund(t.Context(), "pi_1"))

	sent := fake.last()
	assert.Equal(t, http.MethodPost, sent.method)
	assert.Equal(t, "/v1/refunds", sent.path)
	assert.Equal(t, "pi_1", sent.form.Get("payment_intent"))
	assert.Empty(t, sent.form.Get("amount"), "in full")
	assert.Equal(t, "refund-pi_1", sent.header.Get("Idempotency-Key"), "told twice, refunded once")
}

func TestAPaymentRefundedAlreadyIsNoError(t *testing.T) {
	gate, fake := gateway(t, `{"error":{"type":"invalid_request_error","code":"charge_already_refunded",
		"message":"Charge ch_1 has already been refunded."}}`)
	fake.status = http.StatusBadRequest

	require.NoError(t, gate.Refund(t.Context(), "pi_1"))
}

func TestAFailedRefundIsAnError(t *testing.T) {
	gate, fake := gateway(t, `{"error":{"type":"api_error","message":"Try again."}}`)
	fake.status = http.StatusInternalServerError

	require.Error(t, gate.Refund(t.Context(), "pi_1"))
}

func TestASubscriptionIsCancelledNow(t *testing.T) {
	gate, fake := gateway(t, `{"id":"sub_1","object":"subscription","status":"canceled"}`)

	require.NoError(t, gate.Cancel(t.Context(), "sub_1"))

	sent := fake.last()
	assert.Equal(t, http.MethodDelete, sent.method)
	assert.Equal(t, "/v1/subscriptions/sub_1", sent.path)
}

func TestASubscriptionCancelledAlreadyIsNoError(t *testing.T) {
	gate, fake := gateway(t, `{"error":{"type":"invalid_request_error","code":"resource_missing",
		"message":"No such subscription: 'sub_1'"}}`)
	fake.status = http.StatusNotFound

	require.NoError(t, gate.Cancel(t.Context(), "sub_1"))
}
