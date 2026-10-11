package inmemory_note_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/inmemory_note_storage"
)

func TestTheContract(t *testing.T) {
	storage := inmemory_note_storage.New()

	suite.Run(t, &notes.StorageContractSuite{
		NewStorage: func() notes.Storage {
			storage = inmemory_note_storage.New()
			return storage
		},
		Kept: func(notes.Storage) []notes.Snapshot { return storage.Kept() },
	})
}
