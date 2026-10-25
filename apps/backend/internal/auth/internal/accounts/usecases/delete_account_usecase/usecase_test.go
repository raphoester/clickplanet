package delete_account_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/delete_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

func TestTheCallersAccountIsDeletedWithEverythingItHolds(t *testing.T) {
	store := inmemory_account_store.New()
	identity := accounts.NewIdentity("google", accounts.ClaimOf("user", "", false), accounts.AccountID{15: 1}, start)
	session := accounts.LinkedSession(identity.Account(), accounts.TokenOf("token-1"), lifetime, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.NewSignIn(session).WithNewAccount().WithIdentity(identity), &authv1.SignedIn{}))

	seeded := len(store.Told())

	out, err := delete_account_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, &delete_account_usecase.Out{Account: identity.Account(), SetCookie: accounts.ExpiredSessionCookie()}, out)
	_, err = store.Account(t.Context(), identity.Account())
	require.ErrorIs(t, err, accounts.ErrAccountNotFound)
	_, err = store.Identity(t.Context(), "google", "user")
	require.ErrorIs(t, err, accounts.ErrIdentityNotFound)
	told := store.Told()[seeded:]
	require.Len(t, told, 1, "kept with the deletion")
	assert.True(t, proto.Equal(&authv1.AccountDeleted{AccountId: identity.Account().String()}, told[0]))
}

func TestNoCookieDeletesNothing(t *testing.T) {
	store := inmemory_account_store.New()

	out, err := delete_account_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), "theme=dark")

	require.ErrorIs(t, err, accounts.ErrNoAccount)
	assert.Nil(t, out)
	assert.Empty(t, store.Told())
}

func TestAStoreFailureIsNotNoAccount(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := delete_account_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), "cp_sid=token-1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, accounts.ErrNoAccount)
	assert.Empty(t, store.Told(), "nothing was deleted")
}
