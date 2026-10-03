package subscribe_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/inmemory_subscription_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions/usecases/subscribe_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ada = subscriptions.Account{ID: subscriptions.AccountID{15: 1}, Linked: true, Addresses: []subscriptions.Address{"ada@example.com"}}
)

type stubAccounts map[subscriptions.AccountID]subscriptions.Account

func (s stubAccounts) Account(_ context.Context, id subscriptions.AccountID) (subscriptions.Account, error) {
	return s[id], nil
}

type setup struct {
	store    *inmemory_subscription_store.Store
	audience *subscriptions.FakeAudience
	clock    *cptime.FixedClock
	useCase  *subscribe_usecase.UseCase
}

func setUp(accounts ...subscriptions.Account) setup {
	known := stubAccounts{}
	for _, account := range accounts {
		known[account.ID] = account
	}
	s := setup{store: inmemory_subscription_store.New(), audience: &subscriptions.FakeAudience{}, clock: cptime.NewFixedClock(now)}
	s.useCase = subscribe_usecase.New(known, s.store, s.audience, s.clock)
	return s
}

func (s setup) saved(t *testing.T) *subscriptions.Subscription {
	t.Helper()

	found, err := s.store.Subscription(t.Context(), ada.ID)
	require.NoError(t, err)
	return found
}

func TestAVerifiedAddressJoinsTheListAndIsKeptActive(t *testing.T) {
	s := setUp(ada)

	state, err := s.useCase.Execute(t.Context(), ada.ID, " Ada@Example.com ")

	require.NoError(t, err)
	assert.Equal(t, subscriptions.StateActive, state)
	assert.Equal(t, []subscriptions.Call{{Method: "Join", Address: "ada@example.com"}}, s.audience.Calls())
	assert.Equal(t, &subscriptions.Subscription{
		Account: ada.ID, Address: "ada@example.com", State: subscriptions.StateActive,
		Consent: subscriptions.SeasonEmails, AskedAt: now,
	}, s.saved(t))
}

func TestATypedAddressIsInvitedAndKeptWaiting(t *testing.T) {
	s := setUp(ada)

	state, err := s.useCase.Execute(t.Context(), ada.ID, "ada@work.example")

	require.NoError(t, err)
	assert.Equal(t, subscriptions.StateWaiting, state)
	assert.Equal(t, []subscriptions.Call{{Method: "Invite", Address: "ada@work.example"}}, s.audience.Calls())
	assert.Equal(t, subscriptions.StateWaiting, s.saved(t).State)
}

func TestAGuestIsRefusedAndNothingIsAsked(t *testing.T) {
	guest := subscriptions.Account{ID: ada.ID}
	s := setUp(guest)

	_, err := s.useCase.Execute(t.Context(), ada.ID, "ada@example.com")

	require.ErrorIs(t, err, subscriptions.ErrNotLinked)
	assert.Empty(t, s.audience.Calls())
}

func TestAnythingButOneAddressIsRefusedAndNothingIsAsked(t *testing.T) {
	s := setUp(ada)

	_, err := s.useCase.Execute(t.Context(), ada.ID, "ada@example.com, bob@example.com")

	require.ErrorIs(t, err, subscriptions.ErrAddressInvalid)
	assert.Empty(t, s.audience.Calls())
}

func TestAListThatCannotBeReachedKeepsNothing(t *testing.T) {
	s := setUp(ada)
	s.audience.Fail("Join")

	_, err := s.useCase.Execute(t.Context(), ada.ID, "ada@example.com")

	require.ErrorIs(t, err, subscriptions.ErrAudienceUnreachable)
	_, err = s.store.Subscription(t.Context(), ada.ID)
	assert.ErrorIs(t, err, subscriptions.ErrNoSubscription)
}

type failingSaves struct {
	*inmemory_subscription_store.Store
}

func (failingSaves) Save(context.Context, *subscriptions.Subscription) error {
	return errors.New("postgres is down")
}

func TestASubscriptionThatCannotBeKeptTakesTheContactOffTheList(t *testing.T) {
	audience := &subscriptions.FakeAudience{}
	useCase := subscribe_usecase.New(stubAccounts{ada.ID: ada}, failingSaves{inmemory_subscription_store.New()}, audience, cptime.NewFixedClock(now))

	_, err := useCase.Execute(t.Context(), ada.ID, "ada@example.com")

	require.Error(t, err)
	assert.Equal(t, []subscriptions.Call{
		{Method: "Join", Address: "ada@example.com"},
		{Method: "Forget", Address: "ada@example.com"},
	}, audience.Calls())
}

func TestAnotherAddressWhileSubscribedIsRefused(t *testing.T) {
	s := setUp(ada)
	_, err := s.useCase.Execute(t.Context(), ada.ID, "ada@example.com")
	require.NoError(t, err)

	_, err = s.useCase.Execute(t.Context(), ada.ID, "ada@work.example")

	require.ErrorIs(t, err, subscriptions.ErrAlreadySubscribed)
	assert.Len(t, s.audience.Calls(), 1)
	assert.Equal(t, subscriptions.Address("ada@example.com"), s.saved(t).Address)
}

func TestAnOptInAfterAWithdrawalAsksAgainOnTheSameRow(t *testing.T) {
	s := setUp(ada)
	require.NoError(t, s.store.Save(t.Context(), &subscriptions.Subscription{
		Account: ada.ID, Address: "ada@old.example", State: subscriptions.StateWithdrawn,
		Consent: subscriptions.SeasonEmails, AskedAt: now.Add(-48 * time.Hour), WithdrawnAt: now.Add(-24 * time.Hour),
	}))

	state, err := s.useCase.Execute(t.Context(), ada.ID, "ada@example.com")

	require.NoError(t, err)
	assert.Equal(t, subscriptions.StateActive, state)
	assert.Equal(t, &subscriptions.Subscription{
		Account: ada.ID, Address: "ada@example.com", State: subscriptions.StateActive,
		Consent: subscriptions.SeasonEmails, AskedAt: now,
	}, s.saved(t))
}
