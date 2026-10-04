package inmemory_seen_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/inmemory_seen_storage"
)

func TestTheContract(t *testing.T) {
	suite.Run(t, &seen.StorageContractSuite{
		NewStorage: func() seen.Storage { return inmemory_seen_storage.New() },
	})
}
