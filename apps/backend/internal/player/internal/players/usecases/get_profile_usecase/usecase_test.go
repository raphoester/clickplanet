package get_profile_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_profile_usecase"
)

func TestAnAccountThatNeverChoseANameHasAnEmptyOne(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())

	profile := get_profile_usecase.New(storage).Execute(players.AccountID{15: 1})

	assert.Equal(t, players.Profile{Account: players.AccountID{15: 1}}, profile)
}

func TestTheChosenNameIsAnswered(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	saved := players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: time.Date(2026, 9, 17, 0, 0, 0, 0, time.UTC)}
	storage.SaveProfile(saved)

	assert.Equal(t, saved, get_profile_usecase.New(storage).Execute(players.AccountID{15: 1}))
}
