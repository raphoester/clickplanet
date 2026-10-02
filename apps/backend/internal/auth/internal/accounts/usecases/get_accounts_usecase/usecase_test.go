package get_accounts_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/get_accounts_usecase"
)

var start = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestEachKnownAccountIsAnswered(t *testing.T) {
	store := inmemory_account_store.New()
	guest := accounts.GuestSession(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), accounts.Lifetime{}.WithDefaults(), start)
	require.NoError(t, store.CreateGuest(t.Context(), guest))

	found, err := get_accounts_usecase.New(store).Execute(t.Context(), []accounts.AccountID{{15: 1}, {15: 2}})

	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, start, found[0].CreatedAt)
	assert.False(t, found[0].Linked())
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := get_accounts_usecase.New(store).Execute(t.Context(), []accounts.AccountID{{15: 1}})

	assert.Error(t, err)
}
