package get_names_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_names_usecase"
)

func TestOnlyTheAccountsWithANameAreAnswered(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: time.Now()}))
	require.NoError(t, store.RecordTake(t.Context(), players.AccountID{15: 2}, time.Now()))

	names, err := get_names_usecase.New(store).Execute(t.Context(), []players.AccountID{{15: 1}, {15: 2}, {15: 3}})

	require.NoError(t, err)
	assert.Equal(t, map[players.AccountID]players.Name{{15: 1}: "Ada"}, names)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := get_names_usecase.New(store).Execute(t.Context(), []players.AccountID{{15: 1}})

	assert.Error(t, err)
}
