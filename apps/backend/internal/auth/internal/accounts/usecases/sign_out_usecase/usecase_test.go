package sign_out_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/sign_out_usecase"
)

var start = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func TestSigningOutDeletesTheSessionAndClearsTheCookie(t *testing.T) {
	store := inmemory_account_store.New()
	guest := accounts.StartGuest(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), accounts.Lifetime{}.WithDefaults(), start)
	require.NoError(t, store.CreateGuest(t.Context(), guest))

	setCookie, err := sign_out_usecase.New(store).Execute(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, accounts.ClearCookie(), setCookie)
	_, err = store.FindSession(t.Context(), guest.TokenHash)
	assert.ErrorIs(t, err, accounts.ErrSessionNotFound)
}

func TestABrowserWithNoSessionIsSignedOutAlready(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))

	setCookie, err := sign_out_usecase.New(store).Execute(t.Context(), "theme=dark")

	require.NoError(t, err, "nothing to delete, so the store is not asked")
	assert.Equal(t, accounts.ClearCookie(), setCookie)
}

func TestAStoreFailureFailsTheSignOut(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := sign_out_usecase.New(store).Execute(t.Context(), "cp_sid=token-1")

	assert.Error(t, err)
}
