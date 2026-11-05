package delete_account_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/delete_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

func TestASignedInAccountIsSignedOutEverywhereAndKeptForTheGracePeriod(t *testing.T) {
	store := inmemory_account_store.New()
	identity := accounts.NewIdentity("google", accounts.ClaimOf("user", "", false), accounts.AccountID{15: 1}, start)
	session := accounts.LinkedSession(identity.Account(), accounts.TokenOf("token-1"), lifetime, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.NewSignIn(session).WithNewAccount().WithIdentity(identity)))
	other := accounts.LinkedSession(identity.Account(), accounts.TokenOf("token-2"), lifetime, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.NewSignIn(other)))
	outbox := &cpatomic.FakeOutbox{}
	now := start.Add(time.Hour)

	out, err := delete_account_usecase.New(cpatomic.FakeTransactioner{}, store, outbox, cptime.NewFixedClock(now)).
		Execute(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, &delete_account_usecase.Out{Account: identity.Account(), SetCookie: accounts.ExpiredSessionCookie()}, out)
	account, err := store.Account(t.Context(), identity.Account())
	require.NoError(t, err, "kept")
	assert.Equal(t, now, account.DeletionRequestedAt())
	_, err = store.Identity(t.Context(), "google", "user")
	require.NoError(t, err, "a sign-in with it can bring the account back")
	for _, token := range []string{"token-1", "token-2"} {
		_, err = store.Session(t.Context(), accounts.TokenOf(token).Hash())
		require.ErrorIs(t, err, accounts.ErrSessionNotFound, "signed out on every device")
	}
	require.Len(t, outbox.Recorded(), 1, "recorded with the request")
	assert.True(t, proto.Equal(&authv1.AccountDeletionRequested{
		AccountId: identity.Account().String(), RequestedAt: timestamppb.New(now),
	}, outbox.Recorded()[0]))
}

func TestAGuestIsDeletedAtOnceSinceNothingCouldSignInToItAgain(t *testing.T) {
	store := inmemory_account_store.New()
	guest := accounts.GuestSession(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), lifetime, start)
	require.NoError(t, store.CreateGuest(t.Context(), guest))
	outbox := &cpatomic.FakeOutbox{}

	out, err := delete_account_usecase.New(cpatomic.FakeTransactioner{}, store, outbox, cptime.NewFixedClock(start)).
		Execute(t.Context(), "cp_sid=token-1")

	require.NoError(t, err)
	assert.Equal(t, &delete_account_usecase.Out{Account: guest.Account(), SetCookie: accounts.ExpiredSessionCookie()}, out)
	_, err = store.Account(t.Context(), guest.Account())
	require.ErrorIs(t, err, accounts.ErrAccountNotFound)
	require.Len(t, outbox.Recorded(), 1, "recorded with the deletion")
	assert.True(t, proto.Equal(&authv1.AccountDeleted{AccountId: guest.Account().String()}, outbox.Recorded()[0]))
}

func TestNoCookieDeletesNothing(t *testing.T) {
	outbox := &cpatomic.FakeOutbox{}

	out, err := delete_account_usecase.New(cpatomic.FakeTransactioner{}, inmemory_account_store.New(), outbox, cptime.NewFixedClock(start)).
		Execute(t.Context(), "theme=dark")

	require.ErrorIs(t, err, accounts.ErrNoAccount)
	assert.Nil(t, out)
	assert.Empty(t, outbox.Recorded())
}

func TestAStoreFailureIsNotNoAccount(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))
	outbox := &cpatomic.FakeOutbox{}

	_, err := delete_account_usecase.New(cpatomic.FakeTransactioner{}, store, outbox, cptime.NewFixedClock(start)).
		Execute(t.Context(), "cp_sid=token-1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, accounts.ErrNoAccount)
	assert.Empty(t, outbox.Recorded(), "nothing was deleted")
}
