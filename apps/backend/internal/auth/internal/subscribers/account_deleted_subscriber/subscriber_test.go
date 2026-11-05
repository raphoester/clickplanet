package account_deleted_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/inmemory_arrival_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals/usecases/anonymize_arrivals_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/subscribers/account_deleted_subscriber"
)

var at = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestADeletedAccountsArrivalLosesItsAccount(t *testing.T) {
	store := inmemory_arrival_store.New()
	ada := arrivals.AccountID{15: 1}
	store.Seed(arrivals.Snapshot{ID: arrivals.ArrivalID{15: 1}, Account: ada, Source: arrivals.SourceRecruit, Referrer: "discord.com", At: at})

	err := account_deleted_subscriber.New(anonymize_arrivals_usecase.New(store)).Handle(t.Context(),
		&authv1.AccountDeleted{AccountId: ada.String()})

	require.NoError(t, err)
	assert.Equal(t, []arrivals.Snapshot{{ID: arrivals.ArrivalID{15: 1}, Source: arrivals.SourceRecruit, Referrer: "discord.com", At: at}},
		store.Kept())
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := account_deleted_subscriber.New(anonymize_arrivals_usecase.New(inmemory_arrival_store.New())).Handle(t.Context(),
		&authv1.AccountDeleted{})

	require.ErrorIs(t, err, accounts.ErrInvalidAccount)
}
