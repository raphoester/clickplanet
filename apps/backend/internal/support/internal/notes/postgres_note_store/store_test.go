package postgres_note_store_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpmail"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg/cppgtest"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes/postgres_note_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	notes.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppgtest.StartServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	store := postgres_note_store.New(s.db)
	s.NewStorage = func() notes.Storage { return store }
	s.Kept = func(notes.Storage) []notes.Snapshot { return s.kept() }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

func (s *testSuite) kept() []notes.Snapshot {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(),
		`SELECT id, kind, account_id, text, diagnostics, reply_address, submitted_at FROM notes ORDER BY submitted_at, id`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []notes.Snapshot
	for rows.Next() {
		var (
			id      uuid.UUID
			kind    string
			account uuid.NullUUID
			text    string
			said    []byte
			address string
			at      time.Time
		)
		s.Require().NoError(rows.Scan(&id, &kind, &account, &text, &said, &address, &at))
		kept = append(kept, notes.Snapshot{
			ID: notes.NoteID(id), Kind: notes.Kind(kind), Account: notes.AccountID(account.UUID), Text: text,
			Diagnostics: s.diagnosticsOf(said), ReplyAddress: cpmail.Address(address), At: at,
		})
	}
	s.Require().NoError(rows.Err())

	for n := range kept {
		kept[n].Attachments = s.attachmentsOf(kept[n].ID)
	}
	return kept
}

func (s *testSuite) attachmentsOf(note notes.NoteID) []notes.Attachment {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(),
		`SELECT position, id, media_type, size_bytes FROM attachments WHERE note_id = $1 ORDER BY position`, uuid.UUID(note))
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var files []notes.Attachment
	for rows.Next() {
		var (
			position int
			id       uuid.UUID
			media    string
			size     int64
		)
		s.Require().NoError(rows.Scan(&position, &id, &media, &size))
		s.Require().Equal(len(files), position, "the positions count up from 0 with no gap")
		files = append(files, notes.Attachment{ID: notes.AttachmentID(id), MediaType: notes.MediaType(media), Size: size})
	}
	s.Require().NoError(rows.Err())
	return files
}

func (s *testSuite) diagnosticsOf(stored []byte) notes.Diagnostics {
	var read struct {
		Build          string  `json:"build"`
		ViewportWidth  uint32  `json:"viewportWidth"`
		ViewportHeight uint32  `json:"viewportHeight"`
		PixelRatio     float64 `json:"pixelRatio"`
		ConnectionDown bool    `json:"connectionDown"`
		UserAgent      string  `json:"userAgent"`
		GPUVendor      string  `json:"gpuVendor"`
		GPURenderer    string  `json:"gpuRenderer"`
		WebGLVersion   string  `json:"webglVersion"`
		MaxTextureSize uint32  `json:"maxTextureSize"`
		ContextLost    bool    `json:"contextLost"`
		Cores          uint32  `json:"cores"`
		MemoryGB       float64 `json:"memoryGb"`
	}
	decoder := json.NewDecoder(bytes.NewReader(stored))
	decoder.DisallowUnknownFields()
	s.Require().NoError(decoder.Decode(&read), "a key the store writes that this test does not know")
	return notes.Diagnostics(read)
}
