package postgres_decision_store_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/postgres_decision_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/postgres_report_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	ada       = reports.AccountID{15: 1}
	bob       = reports.AccountID{15: 2}
	report    = reports.ReportID{15: 1}
	createdAt = time.Date(2026, 10, 18, 11, 0, 0, 0, time.UTC)
	decidedAt = time.Date(2026, 10, 18, 12, 0, 0, 123_456_000, time.UTC)
)

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	store   *postgres_decision_store.Store
	reports *postgres_report_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	s.store = postgres_decision_store.New(s.db)
	s.reports = postgres_report_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) filed() reports.Report {
	filed, err := reports.NewReport(report, ada, bob, reports.Complaint{Reason: reports.ReasonChatAbuse}, createdAt, nil)
	s.Require().NoError(err)
	s.Require().NoError(s.reports.Save(s.T().Context(), filed))
	return filed
}

func (s *testSuite) decided(outcome decisions.Outcome, comment string, at time.Time) decisions.Decision {
	decision, err := decisions.NewDecision(report, outcome, comment, at)
	s.Require().NoError(err)
	return decision
}

func (s *testSuite) kept() []decisions.Snapshot {
	rows, err := s.db.On(s.T().Context()).QueryContext(s.T().Context(), `SELECT report_id, outcome, comment, decided_at FROM report_decisions`)
	s.Require().NoError(err)
	defer func() { _ = rows.Close() }()

	var kept []decisions.Snapshot
	for rows.Next() {
		var (
			id               uuid.UUID
			outcome, comment string
			at               time.Time
		)
		s.Require().NoError(rows.Scan(&id, &outcome, &comment, &at))
		kept = append(kept, decisions.Snapshot{
			Report: reports.ReportID(id), Outcome: decisions.Outcome(outcome), Comment: comment, At: at.UTC(),
		})
	}
	s.Require().NoError(rows.Err())
	return kept
}

func (s *testSuite) TestADecisionIsKeptWithItsReport() {
	s.filed()

	s.Require().NoError(s.store.Save(s.T().Context(), s.decided(decisions.OutcomeActed, "muted", decidedAt)))

	s.Equal([]decisions.Snapshot{
		{Report: report, Outcome: decisions.OutcomeActed, Comment: "muted", At: decidedAt},
	}, s.kept())
}

func (s *testSuite) TestAReportDecidedAgainKeepsTheLastDecision() {
	s.filed()

	s.Require().NoError(s.store.Save(s.T().Context(), s.decided(decisions.OutcomeDismissed, "", decidedAt)))
	s.Require().NoError(s.store.Save(s.T().Context(), s.decided(decisions.OutcomeActed, "looked again", decidedAt.Add(time.Hour))))

	s.Equal([]decisions.Snapshot{
		{Report: report, Outcome: decisions.OutcomeActed, Comment: "looked again", At: decidedAt.Add(time.Hour)},
	}, s.kept())
}

func (s *testSuite) TestAReportNoLongerKeptCannotBeDecided() {
	err := s.store.Save(s.T().Context(), s.decided(decisions.OutcomeActed, "", decidedAt))

	s.Require().ErrorIs(err, decisions.ErrUnknownReport)
	s.Empty(s.kept())
}

func (s *testSuite) TestADecisionGoesWithItsReport() {
	filed := s.filed()
	s.Require().NoError(s.store.Save(s.T().Context(), s.decided(decisions.OutcomeActed, "", decidedAt)))

	s.Require().NoError(s.reports.Save(s.T().Context(), filed.Forgotten()))

	s.Empty(s.kept())
}
