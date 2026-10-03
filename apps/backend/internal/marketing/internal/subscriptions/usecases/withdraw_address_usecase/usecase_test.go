package withdraw_address_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/inmemory_subscription_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/withdraw_address_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)

func subscription(account byte, address subscriptions.Address, state subscriptions.State) *subscriptions.Subscription {
	return &subscriptions.Subscription{
		Account: subscriptions.AccountID{15: account}, Address: address, State: state,
		Consent: subscriptions.SeasonEmails, AskedAt: now.Add(-time.Hour),
	}
}

func TestEveryLiveRowForTheAddressIsWithdrawnAndTheOthersAreLeft(t *testing.T) {
	store := inmemory_subscription_store.New()
	earlier := subscription(3, "ada@example.com", subscriptions.StateActive).Withdrawn(now.Add(-time.Minute))
	for _, kept := range []*subscriptions.Subscription{
		subscription(1, "ada@example.com", subscriptions.StateActive),
		subscription(2, "ada@example.com", subscriptions.StateWaiting),
		earlier,
		subscription(4, "bob@example.com", subscriptions.StateActive),
	} {
		require.NoError(t, store.Save(t.Context(), kept))
	}

	require.NoError(t, withdraw_address_usecase.New(store, cptime.NewFixedClock(now)).Execute(t.Context(), "ada@example.com"))

	for account, want := range map[byte]*subscriptions.Subscription{
		1: subscription(1, "ada@example.com", subscriptions.StateActive).Withdrawn(now),
		2: subscription(2, "ada@example.com", subscriptions.StateWaiting).Withdrawn(now),
		3: earlier,
		4: subscription(4, "bob@example.com", subscriptions.StateActive),
	} {
		found, err := store.Subscription(t.Context(), subscriptions.AccountID{15: account})
		require.NoError(t, err)
		assert.Equal(t, want, found, "account %d", account)
	}
}

func TestAnAddressNobodyGaveChangesNothing(t *testing.T) {
	assert.NoError(t, withdraw_address_usecase.New(inmemory_subscription_store.New(), cptime.NewFixedClock(now)).
		Execute(t.Context(), "nobody@example.com"))
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_subscription_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, withdraw_address_usecase.New(store, cptime.NewFixedClock(now)).Execute(t.Context(), "ada@example.com"))
}
