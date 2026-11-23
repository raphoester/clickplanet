package inmemory_conversation_storage_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/inmemory_conversation_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, &conversations.StorageContractSuite{
		NewStorage: func() conversations.Storage { return inmemory_conversation_storage.New() },
		KeepNote: func(storage conversations.Storage, note notes.NoteID) {
			storage.(*inmemory_conversation_storage.Storage).KeepNote(note)
		},
	})
}
