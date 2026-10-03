package unsubscribe_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/inmemory_subscription_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/unsubscribe_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now     = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	account = subscriptions.AccountID{15: 1}
	active  = &subscriptions.Subscription{
		Account: account, Address: "ada@example.com", State: subscriptions.StateActive,
		Consent: subscriptions.SeasonEmails, AskedAt: now.Add(-time.Hour),
	}
)

func setUp(t *testing.T, kept ...*subscriptions.Subscription) (*inmemory_subscription_store.Store, *subscriptions.FakeAudience, *unsubscribe_usecase.UseCase) {
	t.Helper()

	store := inmemory_subscription_store.New()
	for _, subscription := range kept {
		require.NoError(t, store.Save(t.Context(), subscription))
	}
	audience := &subscriptions.FakeAudience{}
	return store, audience, unsubscribe_usecase.New(store, audience, cptime.NewFixedClock(now))
}

func TestAnUnsubscribeLeavesTheListAndKeepsTheRowAsWithdrawn(t *testing.T) {
	store, audience, useCase := setUp(t, active)

	require.NoError(t, useCase.Execute(t.Context(), account))

	assert.Equal(t, []subscriptions.Call{{Method: "Leave", Address: "ada@example.com"}}, audience.Calls())
	found, err := store.Subscription(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, active.Withdrawn(now), found)
}

func TestAWaitingSubscriptionIsCancelledTheSameWay(t *testing.T) {
	waiting := *active
	waiting.State = subscriptions.StateWaiting
	store, _, useCase := setUp(t, &waiting)

	require.NoError(t, useCase.Execute(t.Context(), account))

	found, err := store.Subscription(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, subscriptions.StateWithdrawn, found.State)
}

func TestNothingToUnsubscribeAsksNobody(t *testing.T) {
	_, audience, useCase := setUp(t, active.Withdrawn(now.Add(-time.Minute)))

	require.NoError(t, useCase.Execute(t.Context(), account))
	require.NoError(t, useCase.Execute(t.Context(), subscriptions.AccountID{15: 9}))

	assert.Empty(t, audience.Calls())
}

func TestAListThatCannotBeReachedKeepsTheSubscription(t *testing.T) {
	store, audience, useCase := setUp(t, active)
	audience.Fail("Leave")

	err := useCase.Execute(t.Context(), account)

	require.ErrorIs(t, err, subscriptions.ErrAudienceUnreachable)
	found, err := store.Subscription(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, subscriptions.StateActive, found.State)
}
