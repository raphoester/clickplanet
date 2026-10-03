package forget_account_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/inmemory_subscription_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/forget_account_usecase"
)

var (
	account = subscriptions.AccountID{15: 1}
	kept    = &subscriptions.Subscription{
		Account: account, Address: "ada@example.com", State: subscriptions.StateWithdrawn,
		Consent: subscriptions.SeasonEmails, AskedAt: time.Now(), WithdrawnAt: time.Now(),
	}
)

func TestADeletedAccountLeavesBrevoAndLosesItsRowEvenWithdrawn(t *testing.T) {
	store := inmemory_subscription_store.New()
	require.NoError(t, store.Save(t.Context(), kept))
	audience := &subscriptions.FakeAudience{}

	require.NoError(t, forget_account_usecase.New(store, audience).Execute(t.Context(), account))

	assert.Equal(t, []subscriptions.Call{{Method: "Forget", Address: "ada@example.com"}}, audience.Calls())
	_, err := store.Subscription(t.Context(), account)
	assert.ErrorIs(t, err, subscriptions.ErrNoSubscription)
}

func TestAnAccountThatNeverAskedAsksNobody(t *testing.T) {
	audience := &subscriptions.FakeAudience{}

	require.NoError(t, forget_account_usecase.New(inmemory_subscription_store.New(), audience).Execute(t.Context(), account))

	assert.Empty(t, audience.Calls())
}

func TestAContactBrevoStillHoldsKeepsTheRowForTheNextTry(t *testing.T) {
	store := inmemory_subscription_store.New()
	require.NoError(t, store.Save(t.Context(), kept))
	audience := &subscriptions.FakeAudience{}
	audience.Fail("Forget")

	require.Error(t, forget_account_usecase.New(store, audience).Execute(t.Context(), account))

	_, err := store.Subscription(t.Context(), account)
	assert.NoError(t, err)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_subscription_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, forget_account_usecase.New(store, &subscriptions.FakeAudience{}).Execute(t.Context(), account))
}
