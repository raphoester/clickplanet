package postgres_note_store_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
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
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	store := postgres_note_store.New(s.db)
	s.NewStorage = func() notes.Storage { return store }
	s.Kept = func(notes.Storage) []notes.Snapshot { return s.kept() }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

func (s *testSuite) kept() []notes.Snapshot {
	rows, err := s.db.QueryContext(s.T().Context(),
		`SELECT id, kind, account_id, text, diagnostics, submitted_at FROM notes ORDER BY submitted_at, id`)
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
			at      time.Time
		)
		s.Require().NoError(rows.Scan(&id, &kind, &account, &text, &said, &at))
		kept = append(kept, notes.Snapshot{
			ID: notes.NoteID(id), Kind: notes.Kind(kind), Account: notes.AccountID(account.UUID), Text: text,
			Diagnostics: s.diagnosticsOf(said), At: at,
		})
	}
	s.Require().NoError(rows.Err())

	for n := range kept {
		kept[n].Attachments = s.attachmentsOf(kept[n].ID)
	}
	return kept
}

func (s *testSuite) attachmentsOf(note notes.NoteID) []notes.Attachment {
	rows, err := s.db.QueryContext(s.T().Context(),
		`SELECT position, media_type, size_bytes FROM attachments WHERE note_id = $1 ORDER BY position`, uuid.UUID(note))
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var files []notes.Attachment
	for rows.Next() {
		var (
			position int
			media    string
			size     int64
		)
		s.Require().NoError(rows.Scan(&position, &media, &size))
		s.Require().Equal(len(files), position, "the positions count up from 0 with no gap")
		files = append(files, notes.Attachment{MediaType: notes.MediaType(media), Size: size})
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

func (s *testSuite) TestANoteWhosePageSaidNothingKeepsAnObjectOfEmptyValues() {
	note, err := notes.NewSuggestion(notes.NoteID{15: 1}, notes.AccountID{15: 1}, "a night mode", notes.Diagnostics{}, time.Now())
	s.Require().NoError(err)
	s.Require().NoError(postgres_note_store.New(s.db).Save(s.T().Context(), note))

	var build string
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT diagnostics->>'build' FROM notes`).Scan(&build))
	s.Empty(build, "the key is there, with nothing in it")
}

func (s *testSuite) TestTheDatabaseRefusesANoteWithNoDiagnostics() {
	_, err := s.db.ExecContext(s.T().Context(),
		`INSERT INTO notes (id, kind, account_id, text, submitted_at) VALUES ($1, 'suggestion', $2, 'by hand', now())`,
		uuid.New(), uuid.New())

	s.Require().ErrorContains(err, "diagnostics")
}

func (s *testSuite) TestAnAnonymizedNoteHasNoAccountInTheTable() {
	note, err := notes.NewSuggestion(notes.NoteID{15: 1}, notes.AccountID{15: 1}, "a night mode", notes.Diagnostics{}, time.Now())
	s.Require().NoError(err)
	store := postgres_note_store.New(s.db)
	s.Require().NoError(store.Save(s.T().Context(), note))

	s.Require().NoError(store.Save(s.T().Context(), note.Anonymized()))

	var named bool
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT account_id IS NOT NULL FROM notes`).Scan(&named))
	s.False(named, "NULL, not an account of zeros")
}

func (s *testSuite) TestTheDatabaseRefusesAMediaTypeItsTypeDoesNotName() {
	note, err := notes.NewSuggestion(notes.NoteID{15: 1}, notes.AccountID{15: 1}, "a night mode", notes.Diagnostics{}, time.Now())
	s.Require().NoError(err)
	s.Require().NoError(postgres_note_store.New(s.db).Save(s.T().Context(), note))

	_, err = s.db.ExecContext(s.T().Context(),
		`INSERT INTO attachments (note_id, position, media_type, size_bytes) VALUES ($1, 0, 'image/svg+xml', 1)`,
		uuid.UUID{15: 1})

	s.Require().ErrorContains(err, "media_type")
}

func (s *testSuite) TestTheDatabaseRefusesAKindItsTypeDoesNotName() {
	_, err := s.db.ExecContext(s.T().Context(),
		`INSERT INTO notes (id, kind, account_id, text, diagnostics, submitted_at) VALUES ($1, 'complaint', $2, 'by hand', '{}', now())`,
		uuid.New(), uuid.New())

	s.Require().ErrorContains(err, "note_kind")
}
