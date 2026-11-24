package refund_rejected_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/checkouts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/payments/usecases/refund_rejected_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/plans"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
)

const rejected = "00000000-0000-0000-0000-000000000001"

var (
	at    = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	buyer = orders.AccountID{15: 1}
)

type stripe struct {
	expired   []checkouts.SessionID
	refunded  []payments.IntentID
	cancelled []renewals.SubscriptionID
	fails     error
}

func (s *stripe) Expire(_ context.Context, id checkouts.SessionID) error {
	s.expired = append(s.expired, id)
	return nil
}

func (s *stripe) Refund(_ context.Context, intent payments.IntentID) error {
	if s.fails != nil {
		return s.fails
	}
	s.refunded = append(s.refunded, intent)
	return nil
}

func (s *stripe) Cancel(_ context.Context, id renewals.SubscriptionID) error {
	s.cancelled = append(s.cancelled, id)
	return nil
}

func pay(t *testing.T, store *payments.MemoryStore, id payments.PaymentID, intent payments.IntentID, submission string) {
	t.Helper()
	target, err := orders.ParseTarget(submission, "")
	require.NoError(t, err)
	payment, _, err := payments.Pay(payments.Charged{
		ID: id, Order: orders.Snapshot{Buyer: buyer, Plan: plans.Month, Target: target, AgreedAt: at}, Intent: intent,
		Amount: 1000, Currency: "eur", At: at,
	})
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), payment))
}

func TestARejectedFlagIsRefundedInFullAndItsRenewalCancelled(t *testing.T) {
	paid, renewed, gate := payments.NewMemoryStore(), renewals.NewMemoryStore(), &stripe{}
	pay(t, paid, "in_1", "pi_1", rejected)
	pay(t, paid, "in_2", "pi_2", "00000000-0000-0000-0000-000000000002")
	refunded := payments.NewMemoryStore()
	pay(t, refunded, "in_3", "pi_3", rejected)
	taken, err := refunded.Payment(t.Context(), "in_3")
	require.NoError(t, err)
	taken, _, _ = taken.TakenBack(at.Add(time.Minute))
	require.NoError(t, paid.Save(t.Context(), taken))
	ended := renewals.TestRenewing("sub_2", buyer, at.AddDate(0, 1, 0), at)
	ended.State = renewals.Ended
	renewed.Seed(renewals.TestRenewing("sub_1", buyer, at.AddDate(0, 1, 0), at), ended)

	opened := checkouts.NewMemoryStore()
	target, err := orders.ParseTarget(rejected, "")
	require.NoError(t, err)
	require.NoError(t, opened.Save(t.Context(), checkouts.Opened(checkouts.Page{Session: "cs_old"}, target, at.Add(-25*time.Hour))))
	require.NoError(t, opened.Save(t.Context(), checkouts.Opened(checkouts.Page{Session: "cs_open"}, target, at.Add(-time.Minute))))

	require.NoError(t, refund_rejected_usecase.New(cptime.NewFixedClock(at), opened, paid, renewed, gate).Execute(
		t.Context(), rejected))

	assert.Equal(t, []checkouts.SessionID{"cs_open"}, gate.expired, "a page a day old closed on its own")
	assert.Equal(t, []payments.IntentID{"pi_1"}, gate.refunded, "not another flag's, nor one refunded already")
	assert.Equal(t, []renewals.SubscriptionID{"sub_1"}, gate.cancelled, "not one ended already")
}

func TestAFailedRefundIsToldAgain(t *testing.T) {
	paid, gate := payments.NewMemoryStore(), &stripe{fails: errors.New("stripe is down")}
	pay(t, paid, "in_1", "pi_1", rejected)

	err := refund_rejected_usecase.New(cptime.NewFixedClock(at), checkouts.NewMemoryStore(), paid,
		renewals.NewMemoryStore(), gate).Execute(t.Context(), rejected)

	require.Error(t, err, "the event comes again")
}
