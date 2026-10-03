package account_deleted_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscribers/account_deleted_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/inmemory_subscription_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/forget_account_usecase"
)

func TestADeletedAccountIsForgottenHereAndInBrevo(t *testing.T) {
	store := inmemory_subscription_store.New()
	account, err := subscriptions.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), &subscriptions.Subscription{
		Account: account, Address: "ada@example.com", State: subscriptions.StateActive,
		Consent: subscriptions.SeasonEmails, AskedAt: time.Now(),
	}))
	audience := &subscriptions.FakeAudience{}

	err = account_deleted_subscriber.New(forget_account_usecase.New(store, audience)).Handle(t.Context(),
		&authv1.AccountDeleted{AccountId: account.String()})

	require.NoError(t, err)
	assert.Equal(t, []subscriptions.Call{{Method: "Forget", Address: "ada@example.com"}}, audience.Calls())
	_, err = store.Subscription(t.Context(), account)
	assert.ErrorIs(t, err, subscriptions.ErrNoSubscription)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := account_deleted_subscriber.New(forget_account_usecase.New(inmemory_subscription_store.New(), &subscriptions.FakeAudience{})).
		Handle(t.Context(), &authv1.AccountDeleted{})

	assert.ErrorIs(t, err, subscriptions.ErrInvalidAccount)
}
