package inmemory_player_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players/inmemory_player_store"
)

func TestInMemoryStoreHoldsTheContract(t *testing.T) {
	s := new(players.StoreContractSuite)
	s.NewStore = func() players.ReadableStore { return inmemory_player_store.New() }
	suite.Run(t, s)
}
