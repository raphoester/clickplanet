package record_take_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/inmemory_front_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/usecases/record_take_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestATakeCountsForItsFlagAndAgainstTheTilesOwner(t *testing.T) {
	store := inmemory_front_store.New()
	take, err := fronts.NewTake(players.AccountID{15: 1}, "fr", "de")
	require.NoError(t, err)

	require.NoError(t, record_take_usecase.New(store).Execute(t.Context(), take))

	assert.Equal(t, fronts.TallyOf(map[fronts.Country]uint64{"fr": 1}, map[fronts.Country]uint64{"de": 1}),
		store.Tally(players.AccountID{15: 1}))
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_front_store.New()
	store.FailWith(errors.New("postgres is down"))
	take, err := fronts.NewTake(players.AccountID{15: 1}, "fr", "de")
	require.NoError(t, err)

	assert.Error(t, record_take_usecase.New(store).Execute(t.Context(), take))
}
