package name_account_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/name_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	now = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	ada = players.AccountID{15: 1}
)

func useCase(store *inmemory_player_store.Store) *name_account_usecase.UseCase {
	return name_account_usecase.New(players.NewGeneratedNames(store, players.NewRepeatedNames("BraveFox42")), store,
		cptime.NewFixedClock(now))
}

func TestANamelessAccountIsGivenAGeneratedName(t *testing.T) {
	store := inmemory_player_store.New()

	profile, err := useCase(store).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.Name("BraveFox42"), profile.Name)
	assert.True(t, now.Equal(profile.UpdatedAt))
}

func TestAnAccountWithANameKeepsIt(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: ada, Name: "Ada", UpdatedAt: now}))

	profile, err := useCase(store).Execute(t.Context(), ada)

	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), profile.Name)
}

func TestAFailureToNameIsAnError(t *testing.T) {
	store := inmemory_player_store.New()
	refused := errors.New("postgres is down")
	store.FailWith(refused)

	_, err := useCase(store).Execute(t.Context(), ada)

	assert.ErrorIs(t, err, refused)
}
