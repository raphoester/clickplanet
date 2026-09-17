package forget_account_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
)

func TestTheProfileAndTheStatsAreBothForgotten(t *testing.T) {
	storage := inmemory_player_storage.New(inmemory_player_storage.NewMemoryPersistence())
	gone, kept := players.AccountID{15: 1}, players.AccountID{15: 2}
	for _, account := range []players.AccountID{gone, kept} {
		storage.SaveProfile(players.Profile{Account: account, Name: "named", UpdatedAt: time.Now()})
		storage.RecordTake(account, time.Now())
	}

	forget_account_usecase.New(storage).Execute(gone)

	_, named := storage.Profile(gone)
	assert.False(t, named)
	_, played := storage.Stats(gone)
	assert.False(t, played)
	_, played = storage.Stats(kept)
	assert.True(t, played)
}
