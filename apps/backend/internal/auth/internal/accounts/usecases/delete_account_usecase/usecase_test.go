package delete_account_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/delete_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

func TestTheCallersAccountIsDeletedWithEverythingItHolds(t *testing.T) {
	store := inmemory_account_store.New()
	identity := accounts.NewIdentity("google", accounts.Claim{Subject: "user"}, accounts.AccountID{15: 1}, start)
	session := accounts.LinkedSession(identity.Account, accounts.TokenOf("token-1"), lifetime, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.SignIn{NewAccount: true, Identity: identity, Session: session}))

	out, err := delete_account_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, &delete_account_usecase.Out{Account: identity.Account, SetCookie: accounts.ExpiredSessionCookie()}, out)
	_, err = store.Account(t.Context(), identity.Account)
	require.ErrorIs(t, err, accounts.ErrAccountNotFound)
	_, err = store.Identity(t.Context(), "google", "user")
	assert.ErrorIs(t, err, accounts.ErrIdentityNotFound)
}

func TestNoCookieDeletesNothing(t *testing.T) {
	store := inmemory_account_store.New()

	out, err := delete_account_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), "theme=dark")

	require.ErrorIs(t, err, accounts.ErrNoAccount)
	assert.Nil(t, out)
}

func TestAStoreFailureIsNotNoAccount(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := delete_account_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), "cp_sid=token-1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, accounts.ErrNoAccount)
}
