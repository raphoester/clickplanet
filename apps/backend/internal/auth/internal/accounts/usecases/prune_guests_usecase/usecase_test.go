package prune_guests_usecase_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/prune_guests_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

func useCase(store *inmemory_account_store.Store, outbox *cpatomic.FakeOutbox, clock cptime.Clock) *prune_guests_usecase.UseCase {
	return prune_guests_usecase.New(prune_guests_usecase.Config{IdleFor: 90 * 24 * time.Hour}, cpatomic.FakeTransactioner{}, store, outbox, clock)
}

func TestAGuestIsPrunedOnlyOnceItHasBeenIdleTheWholeWindow(t *testing.T) {
	store := inmemory_account_store.New()
	outbox := &cpatomic.FakeOutbox{}
	guest := accounts.GuestSession(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), lifetime, start)
	require.NoError(t, store.CreateGuest(t.Context(), guest))
	clock := cptime.NewFixedClock(start.Add(90*24*time.Hour - time.Second))
	pruned, err := useCase(store, outbox, clock).Execute(t.Context())
	require.NoError(t, err)
	require.Zero(t, pruned)
	require.Empty(t, outbox.Recorded())

	clock.Advance(2 * time.Second)
	pruned, err = useCase(store, outbox, clock).Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)
	_, err = store.Account(t.Context(), guest.Account())
	require.ErrorIs(t, err, accounts.ErrAccountNotFound)
	require.Len(t, outbox.Recorded(), 1, "a pruned guest is a deleted account")
	assert.True(t, proto.Equal(&authv1.AccountDeleted{AccountId: guest.Account().String()}, outbox.Recorded()[0]))
}

func TestALinkedAccountIsNeverPruned(t *testing.T) {
	store := inmemory_account_store.New()
	outbox := &cpatomic.FakeOutbox{}
	identity := accounts.NewIdentity("google", accounts.ClaimOf("user", "", false), accounts.AccountID{15: 1}, start)
	linked := accounts.LinkedSession(identity.Account(), accounts.TokenOf("token-1"), lifetime, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.NewSignIn(linked).WithNewAccount().WithIdentity(identity)))

	pruned, err := useCase(store, outbox, cptime.NewFixedClock(start.Add(365*24*time.Hour))).Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, pruned)
}

func TestAStoreFailureIsReported(t *testing.T) {
	store := inmemory_account_store.New()
	outbox := &cpatomic.FakeOutbox{}
	store.FailWith(errors.New("postgres is down"))

	_, err := useCase(store, outbox, cptime.NewFixedClock(start)).Execute(t.Context())

	assert.Error(t, err)
}
