package inmemory_message_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages/inmemory_message_storage"
)

func TestTheContract(t *testing.T) {
	suite.Run(t, &messages.StorageContractSuite{
		NewStorage: func() messages.Storage { return inmemory_message_storage.New() },
	})
}
