package inmemory_raid_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/inmemory_raid_storage"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &raids.StorageContractSuite{NewStorage: func() raids.Storage { return inmemory_raid_storage.New() }})
}
