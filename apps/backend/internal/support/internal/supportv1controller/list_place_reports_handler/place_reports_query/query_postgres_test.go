package place_reports_query_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cppg"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/postgres_decision_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/migrations"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places/postgres_place_report_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_place_reports_handler/place_reports_query"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type names struct {
	shown map[cpsession.AccountID]string
	err   error
	asked [][]cpsession.AccountID
}

func (n *names) Authors(_ context.Context, accounts []cpsession.AccountID) (map[cpsession.AccountID]*playerv1.Author, error) {
	n.asked = append(n.asked, accounts)
	if n.err != nil {
		return nil, n.err
	}

	found := map[cpsession.AccountID]*playerv1.Author{}
	for _, account := range accounts {
		if name, named := n.shown[account]; named {
			found[account] = &playerv1.Author{AccountId: account.String(), Name: name}
		}
	}
	return found, nil
}

var (
	ada   = places.AccountID{15: 1}
	start = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
)

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	reports *postgres_place_report_store.Store
	names   *names
	query   *place_reports_query.PostgresQuery
}

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	s.reports = postgres_place_report_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.names = &names{shown: map[cpsession.AccountID]string{ada: "Ada_L"}}
	s.query = place_reports_query.NewPostgresQuery(s.db, s.names)
}

func (s *testSuite) reported(n byte, reason places.Reason, text string, at time.Time) *supportv1.PlaceReport {
	id := places.ReportID{15: n}
	report, err := places.NewReport(id, ada, places.TileID(40+uint32(n)), places.Complaint{Reason: reason, Text: text}, at)
	s.Require().NoError(err)
	s.Require().NoError(s.reports.Save(s.T().Context(), report))

	return &supportv1.PlaceReport{
		Id: uuid.UUID(id).String(), ReporterAccountId: ada.String(), ReporterName: "Ada_L",
		TileId: 40 + uint32(n), Text: text, CreatedAt: timestamppb.New(at),
	}
}

func (s *testSuite) decided(n byte, outcome decisions.Outcome, comment string, at time.Time) {
	decision, err := decisions.NewDecision(places.ReportID{15: n}, outcome, comment, at)
	s.Require().NoError(err)
	s.Require().NoError(postgres_decision_store.ForPlaces(s.db).Save(s.T().Context(), decision))
}

func (s *testSuite) read(openOnly bool) []*supportv1.PlaceReport {
	answer, err := s.query.PlaceReports(s.T().Context(), time.Time{}, 0, openOnly)
	s.Require().NoError(err)
	return answer.GetReports()
}

func (s *testSuite) same(want []*supportv1.PlaceReport, got []*supportv1.PlaceReport) {
	s.True(proto.Equal(&supportv1.ListPlaceReportsResponse{Reports: want}, &supportv1.ListPlaceReportsResponse{Reports: got}),
		"want %v, got %v", want, got)
}

func (s *testSuite) TestEveryReportComesNewestFirstWithItsReporterNamedAndItsDecision() {
	first := s.reported(1, places.ReasonHate, "a swastika", start)
	first.Reason = supportv1.PlaceReportReason_PLACE_REPORT_REASON_HATE
	second := s.reported(2, places.ReasonSexual, "", start.Add(time.Minute))
	second.Reason = supportv1.PlaceReportReason_PLACE_REPORT_REASON_SEXUAL
	s.decided(1, decisions.OutcomeActed, "restored the area", start.Add(time.Hour))
	first.Decision = &supportv1.ReportDecision{
		Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_ACTED, Comment: "restored the area",
		DecidedAt: timestamppb.New(start.Add(time.Hour)),
	}

	s.same([]*supportv1.PlaceReport{second, first}, s.read(false))
	s.Equal([][]cpsession.AccountID{{ada}}, s.names.asked, "one ask, each account once")
}

func (s *testSuite) TestTheOpenReportsLeaveOutTheDecidedOnes() {
	s.reported(1, places.ReasonHate, "", start)
	open := s.reported(2, places.ReasonOther, "", start.Add(time.Minute))
	open.Reason = supportv1.PlaceReportReason_PLACE_REPORT_REASON_OTHER
	s.decided(1, decisions.OutcomeDismissed, "", start.Add(time.Hour))

	s.same([]*supportv1.PlaceReport{open}, s.read(true))
}

func (s *testSuite) TestAnAnonymizedReportNamesNobody() {
	report, err := places.NewReport(places.ReportID{15: 1}, ada, 42, places.Complaint{Reason: places.ReasonHate}, start)
	s.Require().NoError(err)
	s.Require().NoError(s.reports.Save(s.T().Context(), report.Anonymized()))

	got := s.read(false)

	s.Require().Len(got, 1)
	s.Empty(got[0].GetReporterAccountId())
	s.Empty(got[0].GetReporterName())
	s.Equal([][]cpsession.AccountID{nil}, s.names.asked)
}

func (s *testSuite) TestAReasonTheDatabaseNamesAndTheWireCannotCarryRefusesTheRead() {
	_, err := s.db.ExecContext(s.T().Context(), `ALTER TYPE place_report_reason ADD VALUE IF NOT EXISTS 'boring'`)
	s.Require().NoError(err)
	_, err = s.db.ExecContext(s.T().Context(), `
		INSERT INTO place_reports (id, reporter_account_id, tile_id, reason, text, created_at)
		VALUES ($1, $2, 42, 'boring', '', $3)
	`, uuid.New(), uuid.UUID(ada), start)
	s.Require().NoError(err)

	_, err = s.query.PlaceReports(s.T().Context(), time.Time{}, 0, false)

	s.Require().ErrorIs(err, place_reports_query.ErrUnknownReason)
}

func (s *testSuite) TestReportsNobodyCouldNameAreRefused() {
	s.reported(1, places.ReasonOther, "", start)
	cause := errors.New("the player module is down")
	s.names.err = cause

	_, err := s.query.PlaceReports(s.T().Context(), time.Time{}, 0, false)

	s.Require().ErrorIs(err, cause)
}
