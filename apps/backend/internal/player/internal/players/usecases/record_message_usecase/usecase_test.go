package record_message_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_message_usecase"
)

func TestEachMessageIsCountedOnTheAccountsStats(t *testing.T) {
	store := inmemory_player_store.New()
	useCase := record_message_usecase.New(store)

	require.NoError(t, useCase.Execute(t.Context(), record_message_usecase.In{Account: players.AccountID{15: 1}}))
	require.NoError(t, useCase.Execute(t.Context(), record_message_usecase.In{Account: players.AccountID{15: 1}}))

	stats, err := store.Stats(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Stats{Account: players.AccountID{15: 1}, MessagesSent: 2}, stats)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))

	err := record_message_usecase.New(store).Execute(t.Context(), record_message_usecase.In{Account: players.AccountID{15: 1}})

	assert.Error(t, err)
}
