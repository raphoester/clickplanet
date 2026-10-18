package postgres_upload_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/postgres_note_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/uploads/postgres_upload_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	uploads.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	store := postgres_upload_store.New(s.db)
	s.NewStorage = func() uploads.Storage { return store }
	s.Send = func(_ uploads.Storage, ids ...notes.AttachmentID) { s.send(ids...) }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

var sentAt = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

// A bug report naming the files, kept by the note store.
func (s *testSuite) send(ids ...notes.AttachmentID) {
	note, err := notes.NewBugReport(notes.NoteID{15: byte(len(ids))}, notes.AccountID{15: 1}, "the globe is black",
		notes.Diagnostics{}, sentAt)
	s.Require().NoError(err)

	files := make([]notes.Attachment, 0, len(ids))
	for _, id := range ids {
		files = append(files, notes.Attachment{ID: id, MediaType: notes.MediaPNG, Size: 1})
	}
	note, err = note.Attaching(files, notes.AttachmentRules{MaxPerNote: 10, MaxBytes: 1000})
	s.Require().NoError(err)

	s.Require().NoError(postgres_note_store.New(s.db).Save(s.T().Context(), note))
}
