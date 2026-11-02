package purge_deletions_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/purge_deletions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
	grace    = 30 * 24 * time.Hour
)

func useCase(store *inmemory_account_store.Store, outbox *cpatomic.FakeOutbox, clock cptime.Clock) *purge_deletions_usecase.UseCase {
	return purge_deletions_usecase.New(purge_deletions_usecase.Config{Grace: grace}, cpatomic.FakeTransactioner{}, store, outbox, clock)
}

func signedIn(t *testing.T, store *inmemory_account_store.Store, account byte) accounts.AccountID {
	t.Helper()
	identity := accounts.NewIdentity("google", accounts.ClaimOf(string(rune('a'+account)), "", false), accounts.AccountID{15: account}, start)
	session := accounts.LinkedSession(identity.Account(), accounts.TokenOf(string(rune('a'+account))), lifetime, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.NewSignIn(session).WithNewAccount().WithIdentity(identity)))
	return identity.Account()
}

func TestAnAccountIsPurgedOnlyOnceTheWholeGracePeriodHasPassed(t *testing.T) {
	store := inmemory_account_store.New()
	outbox := &cpatomic.FakeOutbox{}
	account := signedIn(t, store, 1)
	require.NoError(t, store.RequestDeletion(t.Context(), account, start))
	clock := cptime.NewFixedClock(start.Add(grace - time.Second))

	purged, err := useCase(store, outbox, clock).Execute(t.Context())
	require.NoError(t, err)
	require.Zero(t, purged)
	require.Empty(t, outbox.Recorded())

	clock.Advance(2 * time.Second)
	purged, err = useCase(store, outbox, clock).Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, purged)
	_, err = store.Account(t.Context(), account)
	require.ErrorIs(t, err, accounts.ErrAccountNotFound)
	require.Len(t, outbox.Recorded(), 1, "a purged account is a deleted account")
	assert.True(t, proto.Equal(&authv1.AccountDeleted{AccountId: account.String()}, outbox.Recorded()[0]))
}

func TestAnAccountWhoseDeletionWasNeverRequestedIsNeverPurged(t *testing.T) {
	store := inmemory_account_store.New()
	outbox := &cpatomic.FakeOutbox{}
	signedIn(t, store, 1)

	purged, err := useCase(store, outbox, cptime.NewFixedClock(start.Add(365*24*time.Hour))).Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, purged)
	assert.Empty(t, outbox.Recorded())
}

func TestARestoredAccountIsNotPurged(t *testing.T) {
	store := inmemory_account_store.New()
	outbox := &cpatomic.FakeOutbox{}
	account := signedIn(t, store, 1)
	require.NoError(t, store.RequestDeletion(t.Context(), account, start))
	back := accounts.LinkedSession(account, accounts.TokenOf("back"), lifetime, start.Add(time.Hour))
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.NewSignIn(back).WithRestored()))

	purged, err := useCase(store, outbox, cptime.NewFixedClock(start.Add(2*grace))).Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, purged)
}

func TestAStoreFailureIsReported(t *testing.T) {
	store := inmemory_account_store.New()
	outbox := &cpatomic.FakeOutbox{}
	store.FailWith(errors.New("postgres is down"))

	_, err := useCase(store, outbox, cptime.NewFixedClock(start)).Execute(t.Context())

	assert.Error(t, err)
}
