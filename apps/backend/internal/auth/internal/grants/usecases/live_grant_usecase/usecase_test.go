package live_grant_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/inmemory_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants/usecases/live_grant_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func kept(t *testing.T) (*inmemory_grant_store.Store, grants.Grant) {
	t.Helper()

	store := inmemory_grant_store.New()
	grant, err := grants.NewGrant(grants.GrantID{15: 1}, grants.AccountID{15: 7}, "4242", "the-access-token", start)
	require.NoError(t, err)
	require.NoError(t, store.SaveGrant(t.Context(), grant))
	return store, grant
}

func TestAGrantAnswersUntilItsLifetimeIsOver(t *testing.T) {
	store, grant := kept(t)
	clock := cptime.NewFixedClock(start.Add(grants.Lifetime - time.Nanosecond))

	found, err := live_grant_usecase.New(store, clock).Execute(t.Context(), grant.ID())
	require.NoError(t, err)
	assert.Equal(t, grant.TakeSnapshot(), found.TakeSnapshot())

	clock.Advance(time.Nanosecond)
	_, err = live_grant_usecase.New(store, clock).Execute(t.Context(), grant.ID())
	require.ErrorIs(t, err, grants.ErrGrantExpired)
}

func TestAGrantReadIsReadAgain(t *testing.T) {
	store, grant := kept(t)
	useCase := live_grant_usecase.New(store, cptime.NewFixedClock(start))

	_, err := useCase.Execute(t.Context(), grant.ID())
	require.NoError(t, err)
	_, err = useCase.Execute(t.Context(), grant.ID())

	require.NoError(t, err, "a join tried again reads the grant again")
	assert.Len(t, store.Kept(), 1)
}

func TestAGrantNobodyKeptIsNotFound(t *testing.T) {
	store, _ := kept(t)

	_, err := live_grant_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), grants.GrantID{15: 2})

	require.ErrorIs(t, err, grants.ErrGrantNotFound)
}

func TestAStoreFailureIsNoAnswer(t *testing.T) {
	store, grant := kept(t)
	store.FailWith(errors.New("postgres is down"))

	_, err := live_grant_usecase.New(store, cptime.NewFixedClock(start)).Execute(t.Context(), grant.ID())

	require.Error(t, err)
	assert.NotErrorIs(t, err, grants.ErrGrantNotFound)
}
