package set_name_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestTheCleanedNameIsKept(t *testing.T) {
	store := inmemory_player_store.New()

	profile, err := set_name_usecase.New(store, cptime.NewFixedClock(now)).
		Execute(t.Context(), set_name_usecase.In{Account: players.AccountID{15: 1}, Name: "  Ada\n"})

	require.NoError(t, err)
	want := players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: now}
	assert.Equal(t, want, profile)
	stored, err := store.Profile(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, want, stored)
}

func TestAnInvalidNameChangesNothing(t *testing.T) {
	store := inmemory_player_store.New()
	require.NoError(t, store.SaveProfile(t.Context(), players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: now}))

	_, err := set_name_usecase.New(store, cptime.NewFixedClock(now.Add(time.Hour))).
		Execute(t.Context(), set_name_usecase.In{Account: players.AccountID{15: 1}, Name: " \t "})

	require.ErrorIs(t, err, players.ErrInvalidName)
	stored, err := store.Profile(t.Context(), players.AccountID{15: 1})
	require.NoError(t, err)
	assert.Equal(t, players.Name("Ada"), stored.Name)
}

func TestAStoreFailureIsNotAnInvalidName(t *testing.T) {
	store := inmemory_player_store.New()
	store.FailWith(errors.New("postgres is down"))

	_, err := set_name_usecase.New(store, cptime.NewFixedClock(now)).
		Execute(t.Context(), set_name_usecase.In{Account: players.AccountID{15: 1}, Name: "Ada"})

	require.Error(t, err)
	assert.NotErrorIs(t, err, players.ErrInvalidName)
}
