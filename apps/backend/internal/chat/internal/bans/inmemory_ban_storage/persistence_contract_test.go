package inmemory_ban_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/bans/inmemory_ban_storage"
)

func TestMemoryPersistenceKeepsTheContract(t *testing.T) {
	suite.Run(t, new(memoryPersistenceSuite))
}

type memoryPersistenceSuite struct {
	inmemory_ban_storage.PersistenceContractSuite
}

func (s *memoryPersistenceSuite) SetupSuite() {
	s.NewPersistence = func() inmemory_ban_storage.Persistence {
		return inmemory_ban_storage.NewMemoryPersistence()
	}
}
