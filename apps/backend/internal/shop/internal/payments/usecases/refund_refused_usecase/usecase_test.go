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
)

var at = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)

type stripe struct{ refunded []payments.IntentID }

func (s *stripe) Refund(_ context.Context, intent payments.IntentID) error {
	s.refunded = append(s.refunded, intent)
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

	require.NoError(t, refund_refused_usecase.New(kept(t, "cs_1", "pi_1", false), gate).Execute(t.Context(), "stripe:cs_1"))

	assert.Equal(t, []payments.IntentID{"pi_1"}, gate.refunded)
}

func TestAPaymentRefundedAlreadyAsksNothing(t *testing.T) {
	gate := &stripe{}

	require.NoError(t, refund_refused_usecase.New(kept(t, "cs_1", "pi_1", true), gate).Execute(t.Context(), "stripe:cs_1"))

	assert.Empty(t, gate.refunded)
}

func TestAGrantTheShopNeverMadeIsAnError(t *testing.T) {
	gate := &stripe{}
	useCase := refund_refused_usecase.New(payments.NewMemoryStore(), gate)

	require.ErrorIs(t, useCase.Execute(t.Context(), "operator:1"), payments.ErrInvalidPayment)
	require.ErrorIs(t, useCase.Execute(t.Context(), "stripe:cs_9"), payments.ErrUnknownPayment)
	assert.Empty(t, gate.refunded)
}
