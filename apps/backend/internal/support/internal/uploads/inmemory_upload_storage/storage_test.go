package inmemory_upload_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads/inmemory_upload_storage"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &uploads.StorageContractSuite{
		NewStorage: func() uploads.Storage { return inmemory_upload_storage.New() },
		Send: func(storage uploads.Storage, ids ...notes.AttachmentID) {
			storage.(*inmemory_upload_storage.Storage).Send(ids...)
		},
	})
}
