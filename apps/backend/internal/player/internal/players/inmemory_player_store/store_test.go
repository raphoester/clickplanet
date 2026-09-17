package inmemory_player_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	players.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() players.Store { return inmemory_player_store.New() }
	s.MakeAdmin = func(store players.Store, account players.AccountID) {
		store.(*inmemory_player_store.Store).MakeAdmin(account)
	}
}
