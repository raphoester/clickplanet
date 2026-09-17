package set_name_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)

func TestTheCleanedNameIsKept(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())

	profile, err := set_name_usecase.New(storage, cptime.NewFixedClock(now)).
		Execute(set_name_usecase.In{Account: players.AccountID{15: 1}, Name: "  Ada\n"})

	require.NoError(t, err)
	want := players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: now}
	assert.Equal(t, want, profile)
	stored, _ := storage.Profile(players.AccountID{15: 1})
	assert.Equal(t, want, stored)
}

func TestAnInvalidNameChangesNothing(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	storage.SaveProfile(players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: now})

	_, err := set_name_usecase.New(storage, cptime.NewFixedClock(now.Add(time.Hour))).
		Execute(set_name_usecase.In{Account: players.AccountID{15: 1}, Name: " \t "})

	require.ErrorIs(t, err, players.ErrInvalidName)
	stored, _ := storage.Profile(players.AccountID{15: 1})
	assert.Equal(t, players.Name("Ada"), stored.Name)
}
