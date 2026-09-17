package inmemory_player_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_storage"
)

func TestRunPersistenceContract(t *testing.T) {
	suite.Run(t, new(persistenceSuite))
}

type persistenceSuite struct {
	players.PersistenceContractSuite
}

func (s *persistenceSuite) SetupSuite() {
	s.NewPersistence = func() players.Persistence { return inmemory_player_storage.NewMemoryPersistence() }
}
