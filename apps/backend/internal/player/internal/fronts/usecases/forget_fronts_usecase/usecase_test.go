package forget_fronts_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/inmemory_front_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts/usecases/forget_fronts_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func TestADeletedAccountPlaysForAndAgainstNobody(t *testing.T) {
	store := inmemory_front_store.New()
	take, err := fronts.NewTake(players.AccountID{15: 1}, "fr", "de")
	require.NoError(t, err)
	require.NoError(t, store.RecordTake(t.Context(), take))

	require.NoError(t, forget_fronts_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}))

	assert.Equal(t, fronts.TallyOf(nil, nil), store.Tally(players.AccountID{15: 1}))
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_front_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, forget_fronts_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}))
}
