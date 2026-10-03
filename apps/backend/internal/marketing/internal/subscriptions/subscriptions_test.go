package subscriptions_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/marketing/internal/subscriptions"
)

var (
	now = time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	ada = subscriptions.Account{
		ID: subscriptions.AccountID{15: 1}, Linked: true,
		Addresses: []subscriptions.Address{"ada@example.com", "ada@gmail.com"},
	}
)

func TestAnAddressIsTrimmedAndInLowerCase(t *testing.T) {
	address, err := subscriptions.AddressOf("  Ada.Lovelace+season@Example.COM ")

	require.NoError(t, err)
	assert.Equal(t, subscriptions.Address("ada.lovelace+season@example.com"), address)
}

func TestAnythingButOneBareAddressIsRefused(t *testing.T) {
	for _, raw := range []string{
		"",
		"ada",
		"ada@",
		"@example.com",
		"Ada <ada@example.com>",
		"ada@example.com (comment)",
		`"ada lovelace"@example.com`,
		"ada@localhost",
		"ada@[192.0.2.1]",
		"ada@example..com",
		"ada@-example.com",
		"ada@example.com, bob@example.com",
		strings.Repeat("a", 250) + "@example.com",
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := subscriptions.AddressOf(raw)

			assert.ErrorIs(t, err, subscriptions.ErrAddressInvalid)
		})
	}
}

func TestTheFirstVerifiedAddressFillsTheField(t *testing.T) {
	assert.Equal(t, subscriptions.Address("ada@example.com"), ada.Suggestion())
	assert.Empty(t, subscriptions.Account{Linked: true}.Suggestion())
}

func TestAVerifiedAddressIsActiveAtOnce(t *testing.T) {
	subscription, err := subscriptions.NewSubscription(ada, "ada@gmail.com", now)

	require.NoError(t, err)
	assert.Equal(t, &subscriptions.Subscription{
		Account: ada.ID, Address: "ada@gmail.com", State: subscriptions.StateActive,
		Consent: subscriptions.SeasonEmails, AskedAt: now,
	}, subscription)
}

func TestATypedAddressWaitsForItsConfirmation(t *testing.T) {
	subscription, err := subscriptions.NewSubscription(ada, "ada@work.example", now)

	require.NoError(t, err)
	assert.Equal(t, subscriptions.StateWaiting, subscription.State)
	assert.True(t, subscription.Waiting())
	assert.Equal(t, subscriptions.StateActive, subscription.Confirmed().State)
	assert.Equal(t, subscriptions.StateWaiting, subscription.State, "a confirmation is a copy")
}

func TestAGuestMayNotAsk(t *testing.T) {
	_, err := subscriptions.NewSubscription(subscriptions.Account{ID: ada.ID}, "ada@example.com", now)

	assert.ErrorIs(t, err, subscriptions.ErrNotLinked)
}

func TestTheConsentIsTheVersionOfTheButtonsWords(t *testing.T) {
	assert.Equal(t, "season-emails-1", string(subscriptions.SeasonEmails))
}

func TestAWithdrawalKeepsWhatWasAskedAndSaysWhen(t *testing.T) {
	subscription, err := subscriptions.NewSubscription(ada, "ada@example.com", now)
	require.NoError(t, err)

	withdrawn := subscription.Withdrawn(now.Add(time.Hour))

	assert.Equal(t, subscriptions.StateWithdrawn, withdrawn.State)
	assert.Equal(t, now.Add(time.Hour), withdrawn.WithdrawnAt)
	assert.Equal(t, now, withdrawn.AskedAt)
	assert.Equal(t, subscriptions.SeasonEmails, withdrawn.Consent)
	assert.True(t, subscription.Live())
	assert.False(t, withdrawn.Live())
}

func TestALiveSubscriptionShowsItsOwnAddress(t *testing.T) {
	subscription, err := subscriptions.NewSubscription(ada, "ada@work.example", now)
	require.NoError(t, err)

	assert.Equal(t, subscriptions.Status{State: subscriptions.StateWaiting, Address: "ada@work.example"}, subscription.Status(ada))
}

func TestAWithdrawnOrMissingSubscriptionShowsTheFirstVerifiedAddress(t *testing.T) {
	subscription, err := subscriptions.NewSubscription(ada, "ada@work.example", now)
	require.NoError(t, err)

	none := subscriptions.Status{State: subscriptions.StateNone, Address: "ada@example.com"}
	assert.Equal(t, none, subscription.Withdrawn(now).Status(ada))
	assert.Equal(t, none, subscriptions.StatusOf(ada))
}

func TestAnotherAddressWhileLiveIsRefusedAndTheSameOneIsNot(t *testing.T) {
	subscription, err := subscriptions.NewSubscription(ada, "ada@example.com", now)
	require.NoError(t, err)

	assert.ErrorIs(t, subscription.ChangeError("ada@gmail.com"), subscriptions.ErrAlreadySubscribed)
	require.NoError(t, subscription.ChangeError("ada@example.com"))
	assert.NoError(t, subscription.Withdrawn(now).ChangeError("ada@gmail.com"))
}

func TestAnAccountIDIsAUUIDThatIsNotNil(t *testing.T) {
	account, err := subscriptions.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", account.String())

	for _, raw := range []string{"", "not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		_, err := subscriptions.AccountIDOf(raw)
		assert.ErrorIs(t, err, subscriptions.ErrInvalidAccount, raw)
	}
}
