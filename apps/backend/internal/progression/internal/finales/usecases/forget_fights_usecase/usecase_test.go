package forget_fights_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales/inmemory_fight_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales/usecases/forget_fights_usecase"
)

func TestADeletedAccountLosesItsFights(t *testing.T) {
	store := inmemory_fight_store.New()
	fight, err := finales.FightSnapshot{Account: accounts.AccountID{15: 1}, Season: 0, Country: "fr", Rank: 1}.Restore()
	require.NoError(t, err)
	require.NoError(t, store.Record(t.Context(), []finales.Fight{fight}))

	require.NoError(t, forget_fights_usecase.New(store).Execute(t.Context(), accounts.AccountID{15: 1}))

	found, err := store.Fights(t.Context(), []accounts.AccountID{{15: 1}})
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestAStoreFailureIsAnError(t *testing.T) {
	store := inmemory_fight_store.New()
	store.FailWith(errors.New("postgres is down"))

	assert.Error(t, forget_fights_usecase.New(store).Execute(t.Context(), accounts.AccountID{15: 1}))
}
