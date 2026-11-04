package decide_departure_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures/inmemory_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/departures/usecases/decide_departure_usecase"
)

var (
	start   = time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	account = accounts.AccountID{15: 1}
)

func TestARequestHidesTheAccountUntilALaterRestore(t *testing.T) {
	store := inmemory_departure_store.New()
	useCase := decide_departure_usecase.New(store)

	require.NoError(t, useCase.Execute(t.Context(), account, departures.Leaves(start)))
	left, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.True(t, left.Leaving())

	require.NoError(t, useCase.Execute(t.Context(), account, departures.Returns(start.Add(time.Hour))))
	back, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.False(t, back.Leaving())
}

func TestARequestHeardAfterALaterRestoreLeavesTheAccountShown(t *testing.T) {
	store := inmemory_departure_store.New()
	useCase := decide_departure_usecase.New(store)

	require.NoError(t, useCase.Execute(t.Context(), account, departures.Returns(start.Add(time.Hour))))
	require.NoError(t, useCase.Execute(t.Context(), account, departures.Leaves(start)))

	departure, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.False(t, departure.Leaving())
}

func TestADecisionWithNoTimeIsRefused(t *testing.T) {
	store := inmemory_departure_store.New()

	err := decide_departure_usecase.New(store).Execute(t.Context(), account, departures.Leaves(time.Time{}))

	require.ErrorIs(t, err, departures.ErrNoTime)
	departure, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.False(t, departure.Leaving())
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_departure_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, decide_departure_usecase.New(store).Execute(t.Context(), account, departures.Leaves(start)))
}
