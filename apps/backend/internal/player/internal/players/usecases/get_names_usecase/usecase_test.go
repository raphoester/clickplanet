package get_names_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/get_names_usecase"
)

func TestOnlyTheAccountsWithANameAreAnswered(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	storage.SaveProfile(players.Profile{Account: players.AccountID{15: 1}, Name: "Ada", UpdatedAt: time.Now()})
	storage.RecordTake(players.AccountID{15: 2}, time.Now())

	names := get_names_usecase.New(storage).Execute([]players.AccountID{{15: 1}, {15: 2}, {15: 3}})

	assert.Equal(t, map[players.AccountID]players.Name{{15: 1}: "Ada"}, names)
}
