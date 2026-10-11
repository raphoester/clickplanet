package postgres_report_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/postgres_report_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	reports.StorageContractSuite

	db *cppg.Postgres
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	store := postgres_report_store.New(s.db)
	s.NewStorage = func() reports.Storage { return store }
	s.Kept = func(reports.Storage) []reports.Snapshot { return s.kept() }
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.StorageContractSuite.SetupTest()
}

func (s *testSuite) kept() []reports.Snapshot {
	rows, err := s.db.QueryContext(s.T().Context(), `
		SELECT id, reporter_account_id, reported_account_id, reason, text, message_id, reported_at
		FROM reports ORDER BY reported_at, id
	`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []reports.Snapshot
	for rows.Next() {
		var (
			id, reporter, reported uuid.UUID
			reason, text, message  string
			at                     time.Time
		)
		s.Require().NoError(rows.Scan(&id, &reporter, &reported, &reason, &text, &message, &at))
		kept = append(kept, reports.Snapshot{
			ID: reports.ReportID(id), Reporter: reports.AccountID(reporter), Reported: reports.AccountID(reported), At: at,
			Complaint: reports.Complaint{Reason: reports.Reason(reason), Text: text, Message: reports.MessageID(message)},
		})
	}
	s.Require().NoError(rows.Err())
	return kept
}

func (s *testSuite) TestTheDatabaseRefusesAReasonItsTypeDoesNotName() {
	_, err := s.db.ExecContext(s.T().Context(), `
		INSERT INTO reports (id, reporter_account_id, reported_account_id, reason, text, message_id, reported_at)
		VALUES ($1, $2, $3, 'rude', '', '', now())
	`, uuid.New(), uuid.New(), uuid.New())

	s.Require().ErrorContains(err, "report_reason")
}
