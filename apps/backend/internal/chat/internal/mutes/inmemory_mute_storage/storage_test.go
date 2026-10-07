package inmemory_mute_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/mutes/inmemory_mute_storage"
)

func TestTheContract(t *testing.T) {
	suite.Run(t, &mutes.StorageContractSuite{
		NewStorage: func() mutes.Storage { return inmemory_mute_storage.New() },
	})
}
