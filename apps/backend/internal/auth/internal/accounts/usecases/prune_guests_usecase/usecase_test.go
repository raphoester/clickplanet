package prune_guests_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/inmemory_account_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts/usecases/prune_guests_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	start    = time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	lifetime = accounts.Lifetime{}.WithDefaults()
)

func useCase(store *inmemory_account_store.Store, clock cptime.Clock) *prune_guests_usecase.UseCase {
	return prune_guests_usecase.New(prune_guests_usecase.Config{IdleFor: 90 * 24 * time.Hour}, store, clock)
}

func TestAGuestIsPrunedOnlyOnceItHasBeenIdleTheWholeWindow(t *testing.T) {
	store := inmemory_account_store.New()
	guest := accounts.StartGuest(accounts.AccountID{15: 1}, accounts.TokenOf("token-1"), lifetime, start)
	require.NoError(t, store.CreateGuest(t.Context(), guest))
	clock := cptime.NewFixedClock(start.Add(90*24*time.Hour - time.Second))

	pruned, err := useCase(store, clock).Execute(t.Context())
	require.NoError(t, err)
	require.Zero(t, pruned)

	clock.Advance(2 * time.Second)
	pruned, err = useCase(store, clock).Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)
	_, err = store.FindAccount(t.Context(), guest.Account)
	assert.ErrorIs(t, err, accounts.ErrAccountNotFound)
}

func TestALinkedAccountIsNeverPruned(t *testing.T) {
	store := inmemory_account_store.New()
	identity := accounts.NewIdentity("google", accounts.Claim{Subject: "user"}, accounts.AccountID{15: 1}, start)
	require.NoError(t, store.SaveSignIn(t.Context(), accounts.SignIn{
		NewAccount: true, Identity: identity, Session: accounts.StartLinked(identity.Account, accounts.TokenOf("token-1"), lifetime, start),
	}))

	pruned, err := useCase(store, cptime.NewFixedClock(start.Add(365*24*time.Hour))).Execute(t.Context())

	require.NoError(t, err)
	assert.Zero(t, pruned)
}

func TestAStoreFailureIsReported(t *testing.T) {
	store := inmemory_account_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := useCase(store, cptime.NewFixedClock(start)).Execute(t.Context())

	assert.Error(t, err)
}
