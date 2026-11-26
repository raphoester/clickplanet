package refund_refused_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/refund_refused_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

var at = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

type stripe struct {
	refunded  []payments.IntentID
	cancelled []renewals.SubscriptionID
}

func (s *stripe) Refund(_ context.Context, intent payments.IntentID) error {
	s.refunded = append(s.refunded, intent)
	return nil
}

func (s *stripe) Cancel(_ context.Context, id renewals.SubscriptionID) error {
	s.cancelled = append(s.cancelled, id)
	return nil
}

func kept(t *testing.T, id payments.PaymentID, intent payments.IntentID, takenBack bool) *payments.MemoryStore {
	t.Helper()
	paid := payments.NewMemoryStore()
	payment, _, err := payments.Pay(payments.Charged{
		ID: id, Order: payments.TestOrder(orders.AccountID{15: 1}, plans.Week, at), Intent: intent, Amount: 400,
		Currency: "eur", At: at,
	})
	require.NoError(t, err)
	if takenBack {
		payment, _, _ = payment.TakenBack(at.Add(time.Minute))
	}
	require.NoError(t, paid.Save(t.Context(), payment))
	return paid
}

func TestAPaymentTheFlagsRefusedIsRefundedInFull(t *testing.T) {
	gate := &stripe{}

	require.NoError(t, refund_refused_usecase.New(kept(t, "cs_1", "pi_1", false), renewals.NewMemoryStore(), gate).
		Execute(t.Context(), "stripe:cs_1"))

	assert.Equal(t, []payments.IntentID{"pi_1"}, gate.refunded)
	assert.Empty(t, gate.cancelled)
}

// A late renewal of a flag another player claimed since: the order never buys time again.
func TestTheBuyersRenewalOfARefusedPaymentIsCancelled(t *testing.T) {
	gate := &stripe{}
	paid := kept(t, "in_1", "pi_1", false)
	renewed := renewals.NewMemoryStore()
	payment, err := paid.Payment(t.Context(), "in_1")
	require.NoError(t, err)
	ours := renewals.TestRenewing("sub_1", orders.AccountID{15: 1}, at.AddDate(0, 1, 0), at)
	ours.Order.Target = payment.TakeSnapshot().Order.Target
	theirs := renewals.TestRenewing("sub_2", orders.AccountID{15: 2}, at.AddDate(0, 1, 0), at)
	theirs.Order.Target = payment.TakeSnapshot().Order.Target
	ended := renewals.TestRenewing("sub_3", orders.AccountID{15: 1}, at.AddDate(0, 1, 0), at)
	ended.Order.Target, ended.State = payment.TakeSnapshot().Order.Target, renewals.Ended
	renewed.Seed(ours, theirs, ended)

	require.NoError(t, refund_refused_usecase.New(paid, renewed, gate).Execute(t.Context(), "stripe:in_1"))

	assert.Equal(t, []renewals.SubscriptionID{"sub_1"}, gate.cancelled, "the buyer's own, while it may charge")
	assert.Equal(t, []payments.IntentID{"pi_1"}, gate.refunded)
}

func TestAPaymentRefundedAlreadyAsksNothing(t *testing.T) {
	gate := &stripe{}

	require.NoError(t, refund_refused_usecase.New(kept(t, "cs_1", "pi_1", true), renewals.NewMemoryStore(), gate).
		Execute(t.Context(), "stripe:cs_1"))

	assert.Empty(t, gate.refunded)
}

func TestAGrantTheShopNeverMadeIsAnError(t *testing.T) {
	gate := &stripe{}
	useCase := refund_refused_usecase.New(payments.NewMemoryStore(), renewals.NewMemoryStore(), gate)

	require.ErrorIs(t, useCase.Execute(t.Context(), "operator:1"), payments.ErrInvalidPayment)
	require.ErrorIs(t, useCase.Execute(t.Context(), "stripe:cs_9"), payments.ErrUnknownPayment)
	assert.Empty(t, gate.refunded)
}
