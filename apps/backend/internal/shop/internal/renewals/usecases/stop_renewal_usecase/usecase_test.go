package stop_renewal_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/orders"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/renewals/usecases/stop_renewal_usecase"
)

var (
	now   = time.Date(2026, 11, 24, 9, 0, 0, 0, time.UTC)
	buyer = orders.AccountID{15: 1}
)

type subscriptions struct{ stopped []renewals.SubscriptionID }

func (s *subscriptions) StopAtPeriodEnd(_ context.Context, id renewals.SubscriptionID) error {
	s.stopped = append(s.stopped, id)
	return nil
}

func TestTheBuyerStopsItsRenewalAtStripeAndKeepsThePeriodPaid(t *testing.T) {
	store, stripe := renewals.NewMemoryStore(), &subscriptions{}
	store.Seed(renewals.TestRenewing("sub_1", buyer, now.AddDate(0, 1, 0), now.Add(-time.Hour)))

	stopped, err := stop_renewal_usecase.New(cptime.NewFixedClock(now), store, stripe).Execute(t.Context(),
		stop_renewal_usecase.In{Account: buyer.String(), Linked: true, Renewal: "sub_1"})
	require.NoError(t, err)

	assert.Equal(t, renewals.Stopping, stopped.State)
	assert.Equal(t, now.AddDate(0, 1, 0), stopped.PeriodEndsAt)
	assert.Equal(t, []renewals.SubscriptionID{"sub_1"}, stripe.stopped)
	kept, _ := store.Kept("sub_1")
	assert.Equal(t, renewals.Stopping, kept.State)
}

func TestAnotherAccountStopsNothing(t *testing.T) {
	store, stripe := renewals.NewMemoryStore(), &subscriptions{}
	store.Seed(renewals.TestRenewing("sub_1", buyer, now.AddDate(0, 1, 0), now.Add(-time.Hour)))

	_, err := stop_renewal_usecase.New(cptime.NewFixedClock(now), store, stripe).Execute(t.Context(),
		stop_renewal_usecase.In{Account: orders.AccountID{15: 2}.String(), Linked: true, Renewal: "sub_1"})

	require.ErrorIs(t, err, renewals.ErrNotYours)
	assert.Empty(t, stripe.stopped)
}
