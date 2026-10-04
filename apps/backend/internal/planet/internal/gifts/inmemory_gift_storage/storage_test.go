package inmemory_gift_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/gifts/inmemory_gift_storage"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &gifts.StorageContractSuite{NewStorage: func() gifts.Storage { return inmemory_gift_storage.New() }})
}
