package get_account_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/get_account_usecase"
)

var start = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func setUp(t *testing.T) *inmemory_account_store.Store {
	t.Helper()

	store := inmemory_account_store.New()
	guest := accounts.GuestSession(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), accounts.Lifetime{}.WithDefaults(), start)
	require.NoError(t, store.CreateGuest(t.Context(), guest))
	return store
}

func TestAGuestIsNotLinked(t *testing.T) {
	account, err := get_account_usecase.New(setUp(t)).Execute(t.Context(), accounts.AccountID{15: 1})

	require.NoError(t, err)
	assert.False(t, account.Linked())
}

func TestAnAccountWithAnIdentityIsLinked(t *testing.T) {
	store := setUp(t)
	identity := accounts.NewIdentity("discord", accounts.Claim{Subject: "discord-user"}, accounts.AccountID{15: 1}, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.SignIn{
		Identity: identity, Session: accounts.LinkedSession(identity.Account, accounts.TokenOf("token-2"), accounts.Lifetime{}.WithDefaults(), start),
	}))

	account, err := get_account_usecase.New(store).Execute(t.Context(), accounts.AccountID{15: 1})

	require.NoError(t, err)
	assert.True(t, account.Linked())
}

func TestAnUnknownAccountIsNotFound(t *testing.T) {
	_, err := get_account_usecase.New(setUp(t)).Execute(t.Context(), accounts.AccountID{15: 9})

	assert.ErrorIs(t, err, accounts.ErrAccountNotFound)
}

func TestAStoreFailureIsNotNotFound(t *testing.T) {
	store := setUp(t)
	store.FailWith(errors.New("postgres is down"))

	_, err := get_account_usecase.New(store).Execute(t.Context(), accounts.AccountID{15: 1})

	require.Error(t, err)
	assert.NotErrorIs(t, err, accounts.ErrAccountNotFound)
}
