package sign_out_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/sign_out_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

var start = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)

func TestSigningOutDeletesTheSessionAndClearsTheCookie(t *testing.T) {
	store := inmemory_account_store.New()
	events := cpbootstrap.NewRecordedEvents()
	guest := accounts.GuestSession(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), accounts.Lifetime{}.WithDefaults(), start)
	require.NoError(t, store.CreateGuest(t.Context(), guest))

	setCookie, err := sign_out_usecase.New(store, events).Execute(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, accounts.ExpiredSessionCookie(), setCookie)
	_, err = store.Session(t.Context(), guest.TokenHash)
	require.ErrorIs(t, err, accounts.ErrSessionNotFound)
	require.Len(t, events.Published(), 1)
	assert.True(t, proto.Equal(&authv1.SignedOut{AccountId: guest.Account.String()}, events.Published()[0]))
}

func TestABrowserWithNoSessionIsSignedOutAlready(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))
	events := cpbootstrap.NewRecordedEvents()

	setCookie, err := sign_out_usecase.New(store, events).Execute(t.Context(), "theme=dark")

	require.NoError(t, err, "nothing to delete, so the store is not asked")
	assert.Equal(t, accounts.ExpiredSessionCookie(), setCookie)
	assert.Empty(t, events.Published())
}

func TestASessionAlreadyGoneIsSignedOutAndNamesNoAccount(t *testing.T) {
	events := cpbootstrap.NewRecordedEvents()

	setCookie, err := sign_out_usecase.New(inmemory_account_store.New(), events).Execute(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, accounts.ExpiredSessionCookie(), setCookie)
	assert.Empty(t, events.Published())
}

func TestAStoreFailureFailsTheSignOut(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))
	events := cpbootstrap.NewRecordedEvents()

	_, err := sign_out_usecase.New(store, events).Execute(t.Context(), "cp_sid=token-1")

	require.Error(t, err)
	assert.Empty(t, events.Published())
}
