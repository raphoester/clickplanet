package postgres_conversation_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/conversations/postgres_conversation_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/postgres_note_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	conversations.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	store := postgres_conversation_store.New(s.db)
	noteStore := postgres_note_store.New(s.db)
	s.NewStorage = func() conversations.Storage { return store }
	s.KeepNote = func(_ conversations.Storage, id notes.NoteID) {
		note, err := notes.Snapshot{
			ID: id, Kind: notes.KindBug, Account: notes.AccountID{15: 1}, Text: "the globe is black",
			ReplyAddress: "ada@example.com", At: time.Date(2026, 11, 23, 11, 0, 0, 0, time.UTC),
		}.Restore()
		s.Require().NoError(err)
		s.Require().NoError(noteStore.Save(s.T().Context(), note))
	}
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}
