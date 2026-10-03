package get_subscription_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/inmemory_subscription_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/get_subscription_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/subscribe_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ada = subscriptions.Account{
		ID: subscriptions.AccountID{15: 1}, Linked: true,
		Addresses: []subscriptions.Address{"ada@example.com", "ada@gmail.com"},
	}
)

type stubAccounts struct {
	account subscriptions.Account
	err     error
}

func (s stubAccounts) Account(context.Context, subscriptions.AccountID) (subscriptions.Account, error) {
	return s.account, s.err
}

type setup struct {
	store     *inmemory_subscription_store.Store
	audience  *subscriptions.FakeAudience
	useCase   *get_subscription_usecase.UseCase
	subscribe *subscribe_usecase.UseCase
}

func setUp(accounts stubAccounts) setup {
	s := setup{store: inmemory_subscription_store.New(), audience: &subscriptions.FakeAudience{}}
	s.useCase = get_subscription_usecase.New(accounts, s.store, s.audience)
	s.subscribe = subscribe_usecase.New(accounts, s.store, s.audience, cptime.NewFixedClock(now))
	return s
}

func TestAnAccountThatNeverAskedIsOfferedItsFirstVerifiedAddress(t *testing.T) {
	status, err := setUp(stubAccounts{account: ada}).useCase.Execute(t.Context(), ada.ID)

	require.NoError(t, err)
	assert.Equal(t, subscriptions.Status{State: subscriptions.StateNone, Address: "ada@example.com"}, status)
}

func TestAnAccountWithNoVerifiedAddressIsOfferedAnEmptyField(t *testing.T) {
	status, err := setUp(stubAccounts{account: subscriptions.Account{ID: ada.ID}}).useCase.Execute(t.Context(), ada.ID)

	require.NoError(t, err)
	assert.Equal(t, subscriptions.Status{State: subscriptions.StateNone}, status)
}

func TestAnActiveSubscriptionShowsItsAddressAndAsksNobody(t *testing.T) {
	s := setUp(stubAccounts{account: ada})
	_, err := s.subscribe.Execute(t.Context(), ada.ID, "ada@gmail.com")
	require.NoError(t, err)

	status, err := s.useCase.Execute(t.Context(), ada.ID)

	require.NoError(t, err)
	assert.Equal(t, subscriptions.Status{State: subscriptions.StateActive, Address: "ada@gmail.com"}, status)
	assert.Len(t, s.audience.Calls(), 1, "only the join")
}

func TestATypedAddressWaitsThenTurnsActiveOnceItJoined(t *testing.T) {
	s := setUp(stubAccounts{account: ada})
	state, err := s.subscribe.Execute(t.Context(), ada.ID, "ada@work.example")
	require.NoError(t, err)
	require.Equal(t, subscriptions.StateWaiting, state)

	status, err := s.useCase.Execute(t.Context(), ada.ID)
	require.NoError(t, err)
	assert.Equal(t, subscriptions.Status{State: subscriptions.StateWaiting, Address: "ada@work.example"}, status)

	s.audience.Confirm("ada@work.example")

	status, err = s.useCase.Execute(t.Context(), ada.ID)
	require.NoError(t, err)
	assert.Equal(t, subscriptions.Status{State: subscriptions.StateActive, Address: "ada@work.example"}, status)
	kept, err := s.store.Subscription(t.Context(), ada.ID)
	require.NoError(t, err)
	assert.Equal(t, subscriptions.StateActive, kept.State)
	assert.Equal(t, now, kept.AskedAt, "a confirmation keeps the time the player asked")
}

func TestAListThatCannotAnswerLeavesTheRowWaiting(t *testing.T) {
	s := setUp(stubAccounts{account: ada})
	_, err := s.subscribe.Execute(t.Context(), ada.ID, "ada@work.example")
	require.NoError(t, err)
	s.audience.Fail("Joined")

	status, err := s.useCase.Execute(t.Context(), ada.ID)

	require.NoError(t, err)
	assert.Equal(t, subscriptions.StateWaiting, status.State)
}

func TestAWithdrawnSubscriptionIsOfferedTheFirstVerifiedAddressAgain(t *testing.T) {
	s := setUp(stubAccounts{account: ada})
	require.NoError(t, s.store.Save(t.Context(), &subscriptions.Subscription{
		Account: ada.ID, Address: "ada@work.example", State: subscriptions.StateWithdrawn,
		Consent: subscriptions.SeasonEmails, AskedAt: now, WithdrawnAt: now,
	}))

	status, err := s.useCase.Execute(t.Context(), ada.ID)

	require.NoError(t, err)
	assert.Equal(t, subscriptions.Status{State: subscriptions.StateNone, Address: "ada@example.com"}, status)
}

func TestAStoreOrAuthFailureIsAnError(t *testing.T) {
	s := setUp(stubAccounts{account: ada})
	s.store.FailWith(errors.New("postgres is down"))
	_, err := s.useCase.Execute(t.Context(), ada.ID)
	require.Error(t, err)

	_, err = setUp(stubAccounts{err: errors.New("auth is down")}).useCase.Execute(t.Context(), ada.ID)
	assert.Error(t, err)
}
