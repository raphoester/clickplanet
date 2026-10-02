package get_creation_dates_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/get_creation_dates_usecase"
)

var start = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestEachKnownAccountSaysWhenItWasMade(t *testing.T) {
	store := inmemory_account_store.New()
	guest := accounts.GuestSession(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), accounts.Lifetime{}.WithDefaults(), start)
	require.NoError(t, store.CreateGuest(t.Context(), guest))

	dates, err := get_creation_dates_usecase.New(store).Execute(t.Context(), []accounts.AccountID{{15: 1}, {15: 2}})

	require.NoError(t, err)
	assert.Equal(t, map[accounts.AccountID]time.Time{{15: 1}: start}, dates)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := get_creation_dates_usecase.New(store).Execute(t.Context(), []accounts.AccountID{{15: 1}})

	assert.Error(t, err)
}
