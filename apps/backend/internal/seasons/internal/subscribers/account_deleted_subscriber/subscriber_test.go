package account_deleted_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/subscribers/account_deleted_subscriber"
)

func TestADeletedAccountLeavesTheStandings(t *testing.T) {
	store := inmemory_contribution_store.New()
	account, err := standings.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, store.RecordTake(t.Context(), 0, standings.Take{Account: account, Country: "fr"}))

	err = account_deleted_subscriber.New(forget_account_usecase.New(store)).Handle(t.Context(),
		&authv1.AccountDeleted{AccountId: account.String()})

	require.NoError(t, err)
	_, err = store.Line(t.Context(), 0, account)
	assert.ErrorIs(t, err, standings.ErrNoLine)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := account_deleted_subscriber.New(forget_account_usecase.New(inmemory_contribution_store.New())).
		Handle(t.Context(), &authv1.AccountDeleted{})

	assert.ErrorIs(t, err, standings.ErrInvalidAccount)
}
