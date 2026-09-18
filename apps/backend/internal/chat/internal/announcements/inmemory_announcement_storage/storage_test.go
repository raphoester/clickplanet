package inmemory_announcement_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements/inmemory_announcement_storage"
)

func TestTheContract(t *testing.T) {
	suite.Run(t, &announcements.StorageContractSuite{
		NewStorage: func() announcements.Storage { return inmemory_announcement_storage.New() },
	})
}
