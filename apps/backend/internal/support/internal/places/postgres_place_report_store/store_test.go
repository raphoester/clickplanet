package postgres_place_report_store_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places/postgres_place_report_store"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	ada   = places.AccountID{15: 1}
	bob   = places.AccountID{15: 2}
	start = time.Date(2026, 10, 20, 12, 0, 0, 123_456_000, time.UTC)
)

type testSuite struct {
	suite.Suite

	db    *cppg.Postgres
	store *postgres_place_report_store.Store
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	s.store = postgres_place_report_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
}

func (s *testSuite) filed(n byte, reporter places.AccountID, reason places.Reason, at time.Time) places.Report {
	report, err := places.NewReport(places.ReportID{15: n}, reporter, 42, places.Complaint{Reason: reason, Text: "a swastika"}, at)
	s.Require().NoError(err)
	s.Require().NoError(s.store.Save(s.T().Context(), report))
	return report
}

func snapshots(reports []places.Report) []places.Snapshot {
	kept := make([]places.Snapshot, 0, len(reports))
	for _, report := range reports {
		kept = append(kept, report.TakeSnapshot())
	}
	return kept
}

func (s *testSuite) TestAReportIsReadBackAsItWasFiledByItsReporter() {
	first := s.filed(1, ada, places.ReasonHate, start)
	second := s.filed(2, ada, places.ReasonSexual, start.Add(time.Minute))
	s.filed(3, bob, places.ReasonOther, start)

	filed, err := s.store.FiledBy(s.T().Context(), ada)
	s.Require().NoError(err)

	s.Equal(snapshots([]places.Report{first, second}), snapshots(filed))
}

func (s *testSuite) TestEveryReasonIsKept() {
	for n, reason := range []places.Reason{places.ReasonHate, places.ReasonSexual, places.ReasonOther} {
		s.filed(byte(n+1), ada, reason, start.Add(time.Duration(n)*time.Second))
	}

	filed, err := s.store.FiledBy(s.T().Context(), ada)
	s.Require().NoError(err)

	s.Len(filed, 3)
}

func (s *testSuite) TestAnAnonymizedReportIsKeptWithNoReporter() {
	report := s.filed(1, ada, places.ReasonHate, start)

	s.Require().NoError(s.store.Save(s.T().Context(), report.Anonymized()))

	filed, err := s.store.FiledBy(s.T().Context(), ada)
	s.Require().NoError(err)
	s.Empty(filed)

	var reporter *string
	s.Require().NoError(s.db.QueryRowContext(s.T().Context(), `SELECT reporter_account_id::text FROM place_reports`).Scan(&reporter))
	s.Nil(reporter, "the row stays, with no account")
}

func (s *testSuite) TestTheReportsPastACutoffWithAReporterAreFoundOldestFirst() {
	old := s.filed(1, ada, places.ReasonHate, start)
	older := s.filed(2, bob, places.ReasonHate, start.Add(-time.Hour))
	s.filed(3, ada, places.ReasonHate, start.Add(time.Hour))
	s.Require().NoError(s.store.Save(s.T().Context(), s.filed(4, bob, places.ReasonOther, start.Add(-2*time.Hour)).Anonymized()))

	found, err := s.store.WithReporterBefore(s.T().Context(), start.Add(time.Minute), 10)
	s.Require().NoError(err)
	s.Equal(snapshots([]places.Report{older, old}), snapshots(found))

	limited, err := s.store.WithReporterBefore(s.T().Context(), start.Add(time.Minute), 1)
	s.Require().NoError(err)
	s.Equal(snapshots([]places.Report{older}), snapshots(limited))
}
