package postgres_note_store_test

import (
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
		`SELECT id, kind, account_id, text, submitted_at FROM notes ORDER BY submitted_at, id`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []notes.Snapshot
	for rows.Next() {
		var (
			id      uuid.UUID
			kind    string
			account uuid.UUID
			text    string
			at      time.Time
		)
		s.Require().NoError(rows.Scan(&id, &kind, &account, &text, &at))
		kept = append(kept, notes.Snapshot{
			ID: notes.NoteID(id), Kind: notes.Kind(kind), Account: notes.AccountID(account), Text: text, At: at,
		})
	}
	s.Require().NoError(rows.Err())
	return kept
}

func (s *testSuite) TestTheDatabaseRefusesAKindItsTypeDoesNotName() {
	_, err := s.db.ExecContext(s.T().Context(),
		`INSERT INTO notes (id, kind, account_id, text, submitted_at) VALUES ($1, 'complaint', $2, 'by hand', now())`,
		uuid.New(), uuid.New())

	s.Require().ErrorContains(err, "note_kind")
}
