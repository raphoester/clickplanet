package record_allegiance_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/record_allegiance_usecase"
)

var at = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

type fakeAllegiances struct {
	tallies map[clicks.AllegianceKey]clicks.Allegiance
	err     error
}

func (f *fakeAllegiances) Allegiances(
	_ context.Context, keys ...clicks.AllegianceKey,
) (map[clicks.AllegianceKey]clicks.Allegiance, error) {
	found := map[clicks.AllegianceKey]clicks.Allegiance{}
	for _, key := range keys {
		if tally, ok := f.tallies[key]; ok {
			found[key] = tally
		}
	}
	return found, f.err
}

func (f *fakeAllegiances) SaveAllegiances(_ context.Context, tallies map[clicks.AllegianceKey]clicks.Allegiance) error {
	for key, tally := range tallies {
		f.tallies[key] = tally
	}
	return f.err
}

func TestATakeCountsForItsAccountAndItsScope(t *testing.T) {
	store := &fakeAllegiances{tallies: map[clicks.AllegianceKey]clicks.Allegiance{}}
	useCase := record_allegiance_usecase.New(store)

	for range 3 {
		require.NoError(t, useCase.Execute(t.Context(), record_allegiance_usecase.In{Account: "ada", Scope: "home", Country: "fr", At: at}))
	}
	require.NoError(t, useCase.Execute(t.Context(), record_allegiance_usecase.In{Account: "ada", Scope: "home", Country: "es", At: at}))

	assert.Equal(t, "fr", store.tallies[clicks.AccountAllegianceKey("ada")].Flag(), "one take for Spain does not move it")
	assert.Equal(t, "fr", store.tallies[clicks.ScopeAllegianceKey("home")].Flag())
}

func TestAFailedReadSavesNothing(t *testing.T) {
	store := &fakeAllegiances{tallies: map[clicks.AllegianceKey]clicks.Allegiance{}, err: errors.New("down")}

	err := record_allegiance_usecase.New(store).Execute(t.Context(), record_allegiance_usecase.In{Account: "ada", Country: "fr", At: at})

	require.Error(t, err)
	assert.Empty(t, store.tallies)
}
