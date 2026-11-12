package forget_departure_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/inmemory_departure_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures/usecases/forget_departure_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestADeletedAccountLeavesNoDepartureBehind(t *testing.T) {
	store := inmemory_departure_store.New()
	account := players.AccountID{15: 1}
	require.NoError(t, store.Change(t.Context(), account, func(d departures.Departure) departures.Departure {
		return d.Decided(departures.Leaves(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)))
	}))

	require.NoError(t, forget_departure_usecase.New(store).Execute(t.Context(), account))

	departure, err := store.Departure(t.Context(), account)
	require.NoError(t, err)
	assert.Equal(t, departures.Departure{}, departure)
}
