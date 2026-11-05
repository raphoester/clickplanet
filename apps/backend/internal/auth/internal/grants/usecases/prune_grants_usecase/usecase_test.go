package prune_grants_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/inmemory_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/usecases/prune_grants_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func keep(t *testing.T, store *inmemory_grant_store.Store, id byte, at time.Time) {
	t.Helper()

	grant, err := grants.NewGrant(grants.GrantID{15: id}, grants.AccountID{15: 7}, "4242", "the-access-token", at)
	require.NoError(t, err)
	require.NoError(t, store.SaveGrant(t.Context(), grant))
}

func TestAGrantIsPrunedOnceItsLifetimeIsOver(t *testing.T) {
	store := inmemory_grant_store.New()
	keep(t, store, 1, start)
	keep(t, store, 2, start.Add(time.Minute))
	clock := cptime.NewFixedClock(start.Add(grants.Lifetime - time.Nanosecond))

	pruned, err := prune_grants_usecase.New(store, clock).Execute(t.Context())
	require.NoError(t, err)
	require.Zero(t, pruned)

	clock.Advance(time.Nanosecond)
	pruned, err = prune_grants_usecase.New(store, clock).Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)
	require.Len(t, store.Kept(), 1)
	assert.Equal(t, grants.GrantID{15: 2}, store.Kept()[0].ID)
}

func TestAStoreFailureIsReported(t *testing.T) {
	store := inmemory_grant_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := prune_grants_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context())

	assert.Error(t, err)
}
