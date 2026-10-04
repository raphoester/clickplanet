package inmemory_take_store_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes/inmemory_take_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	takes.StoreContractSuite
}

func (s *testSuite) SetupSuite() {
	s.NewStore = func() (takes.Store, takes.Players) {
		players := inmemory_player_store.New()
		return inmemory_take_store.New(players), players
	}
}
