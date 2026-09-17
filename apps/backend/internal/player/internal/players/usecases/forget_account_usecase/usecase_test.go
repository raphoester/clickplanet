package forget_account_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
)

func TestTheProfileAndTheStatsAreBothForgotten(t *testing.T) {
	store := inmemory_player_store.New()
	gone, kept := players.AccountID{15: 1}, players.AccountID{15: 2}
	for _, account := range []players.AccountID{gone, kept} {
		require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: account, Name: "named", UpdatedAt: time.Now()}))
		require.NoError(t, store.RecordTake(t.Context(), account, time.Now()))
	}

	require.NoError(t, forget_account_usecase.New(store).Execute(t.Context(), gone))

	_, err := store.Profile(t.Context(), gone)
	require.ErrorIs(t, err, players.ErrNoProfile)
	_, err = store.Stats(t.Context(), gone)
	require.ErrorIs(t, err, players.ErrNoStats)
	_, err = store.Stats(t.Context(), kept)
	assert.NoError(t, err)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, forget_account_usecase.New(store).Execute(t.Context(), players.AccountID{15: 1}))
}
