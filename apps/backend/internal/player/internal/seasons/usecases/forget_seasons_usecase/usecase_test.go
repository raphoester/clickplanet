package forget_seasons_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/inmemory_season_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/seasons/usecases/forget_seasons_usecase"
)

var ada = players.AccountID{15: 1}

func TestADeletedAccountLosesTheSeasonsItPlayed(t *testing.T) {
	store := inmemory_season_store.New()
	require.NoError(t, store.Save(t.Context(), seasons.NewRecord(ada).With(seasons.Season{Number: 0, Flag: "fr"})))

	require.NoError(t, forget_seasons_usecase.New(store).Execute(t.Context(), ada))

	record, err := store.Record(t.Context(), ada)
	require.NoError(t, err)
	assert.False(t, record.Played(0))
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_season_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, forget_seasons_usecase.New(store).Execute(t.Context(), ada))
}
