package record_allegiance_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_allegiance_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/record_allegiance_usecase"
)

var at = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

var (
	ada  = clicks.AccountAllegianceKey("ada")
	home = clicks.ScopeAllegianceKey("home")
)

func TestATakeCountsForItsAccountAndItsScope(t *testing.T) {
	store := inmemory_allegiance_store.New()
	useCase := record_allegiance_usecase.New(store)

	for range 3 {
		require.NoError(t, useCase.Execute(t.Context(), record_allegiance_usecase.In{Account: "ada", Scope: "home", Country: "fr", At: at}))
	}
	require.NoError(t, useCase.Execute(t.Context(), record_allegiance_usecase.In{Account: "ada", Scope: "home", Country: "es", At: at}))

	tallies, err := store.Allegiances(t.Context(), ada, home)
	require.NoError(t, err)
	assert.Equal(t, "fr", tallies[ada].Flag(), "one take for Spain does not move it")
	assert.Equal(t, "fr", tallies[home].Flag())
}

func TestATakeWithNoAccountCountsForItsScopeAlone(t *testing.T) {
	store := inmemory_allegiance_store.New()

	require.NoError(t, record_allegiance_usecase.New(store).Execute(t.Context(),
		record_allegiance_usecase.In{Scope: "home", Country: "fr", At: at}))

	tallies, err := store.Allegiances(t.Context(), ada, home)
	require.NoError(t, err)
	assert.Equal(t, []clicks.AllegianceKey{home}, keysOf(tallies))
}

func TestAFailingStoreFailsTheTake(t *testing.T) {
	store := inmemory_allegiance_store.New()
	cause := errors.New("down")
	store.FailWith(cause)

	err := record_allegiance_usecase.New(store).Execute(t.Context(), record_allegiance_usecase.In{Account: "ada", Country: "fr", At: at})

	require.ErrorIs(t, err, cause)
}

func keysOf(tallies map[clicks.AllegianceKey]clicks.Allegiance) []clicks.AllegianceKey {
	keys := make([]clicks.AllegianceKey, 0, len(tallies))
	for key := range tallies {
		keys = append(keys, key)
	}
	return keys
}
