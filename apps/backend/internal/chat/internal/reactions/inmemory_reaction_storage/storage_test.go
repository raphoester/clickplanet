package inmemory_reaction_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions/inmemory_reaction_storage"
)

func TestTheContract(t *testing.T) {
	suite.Run(t, &reactions.StorageContractSuite{
		NewStorage: func() reactions.Storage { return inmemory_reaction_storage.New() },
	})
}
