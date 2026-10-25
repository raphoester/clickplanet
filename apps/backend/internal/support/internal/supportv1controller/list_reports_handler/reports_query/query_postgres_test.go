package reports_query_test

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
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/postgres_report_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/list_reports_handler/reports_query"
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

type testSuite struct {
	suite.Suite

	db      *cppg.Postgres
	reports *postgres_report_store.Store
	names   *names
	query   *reports_query.PostgresQuery
}

var (
	ada   = reports.AccountID{15: 1}
	bob   = reports.AccountID{15: 2}
	gone  = reports.AccountID{15: 3}
	start = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
)

func (s *testSuite) SetupSuite() {
	s.db = cppg.StartTestServer(s.T()).OpenSchema(s.T(), "support", migrations.FS)
	s.reports = postgres_report_store.New(s.db)
}

func (s *testSuite) SetupTest() {
	s.Require().NoError(s.db.Purge(s.T().Context()))
	s.names = &names{shown: map[cpsession.AccountID]string{ada: "Ada_L", bob: "guest_a1b2c3"}}
	s.query = reports_query.NewPostgresQuery(s.db, s.names)
}

func (s *testSuite) reported(n byte, reporter, about reports.AccountID, complaint reports.Complaint, at time.Time) *supportv1.Report {
	id := reports.ReportID{15: n}
	report, err := reports.NewReport(id, reporter, about, complaint, at, nil)
	s.Require().NoError(err)
	s.Require().NoError(s.reports.Save(s.T().Context(), report))

	return &supportv1.Report{
		Id:                uuid.UUID(id).String(),
		ReporterAccountId: reporter.String(),
		ReportedAccountId: about.String(),
		Text:              complaint.Text,
		MessageId:         string(complaint.Message),
		CreatedAt:         timestamppb.New(at),
		ReporterName:      s.names.shown[reporter],
		ReportedName:      s.names.shown[about],
	}
}

func (s *testSuite) read(since time.Time, limit uint32) []*supportv1.Report {
	answer, err := s.query.Reports(s.T().Context(), since, limit, false)
	s.Require().NoError(err)
	return answer.GetReports()
}

func (s *testSuite) same(want []*supportv1.Report, got []*supportv1.Report) {
	s.True(proto.Equal(&supportv1.ListReportsResponse{Reports: want}, &supportv1.ListReportsResponse{Reports: got}),
		"want %v, got %v", want, got)
}

func (s *testSuite) TestNoReportIsAnEmptyAnswerAndNobodyIsAsked() {
	s.Empty(s.read(time.Time{}, 0))
	s.Equal([][]cpsession.AccountID{nil}, s.names.asked)
}

func (s *testSuite) TestEveryReportComesNewestFirstWithBothAccountsNamed() {
	first := s.reported(1, ada, bob, reports.Complaint{Reason: reports.ReasonChatAbuse, Text: "it insults\neveryone", Message: "message-1"}, start)
	first.Reason = supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE
	second := s.reported(2, bob, ada, reports.Complaint{Reason: reports.ReasonCheating}, start.Add(time.Minute))
	second.Reason = supportv1.ReportReason_REPORT_REASON_CHEATING

	s.same([]*supportv1.Report{second, first}, s.read(time.Time{}, 0))
	s.Len(s.names.asked, 1, "one ask for everyone")
	s.ElementsMatch([]cpsession.AccountID{ada, bob}, s.names.asked[0], "each account once")
}

func (s *testSuite) TestAnAccountNobodyCanNameIsLeftWithNoName() {
	report := s.reported(1, ada, gone, reports.Complaint{Reason: reports.ReasonBadName}, start)
	report.Reason = supportv1.ReportReason_REPORT_REASON_BAD_NAME

	got := s.read(time.Time{}, 0)

	s.same([]*supportv1.Report{report}, got)
	s.Empty(got[0].GetReportedName())
	s.Equal("Ada_L", got[0].GetReporterName())
}

func (s *testSuite) TestEveryReasonTheDomainKnowsIsRead() {
	for n, reason := range []reports.Reason{reports.ReasonCheating, reports.ReasonChatAbuse, reports.ReasonBadName, reports.ReasonOther} {
		s.reported(byte(n+1), ada, bob, reports.Complaint{Reason: reason}, start.Add(time.Duration(n)*24*time.Hour))
	}

	got := s.read(time.Time{}, 0)

	s.Require().Len(got, 4)
	s.Equal([]supportv1.ReportReason{
		supportv1.ReportReason_REPORT_REASON_OTHER, supportv1.ReportReason_REPORT_REASON_BAD_NAME,
		supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE, supportv1.ReportReason_REPORT_REASON_CHEATING,
	}, []supportv1.ReportReason{got[0].GetReason(), got[1].GetReason(), got[2].GetReason(), got[3].GetReason()})
}

func (s *testSuite) TestReportsBeforeSinceAreLeftOutAndTheLimitKeepsTheNewest() {
	s.reported(1, ada, bob, reports.Complaint{Reason: reports.ReasonOther, Text: "old"}, start)
	atSince := s.reported(2, bob, ada, reports.Complaint{Reason: reports.ReasonOther, Text: "at since"}, start.Add(time.Hour))
	atSince.Reason = supportv1.ReportReason_REPORT_REASON_OTHER
	later := s.reported(3, ada, gone, reports.Complaint{Reason: reports.ReasonOther, Text: "later"}, start.Add(2*time.Hour))
	later.Reason = supportv1.ReportReason_REPORT_REASON_OTHER

	s.same([]*supportv1.Report{later, atSince}, s.read(start.Add(time.Hour), 0))
	s.same([]*supportv1.Report{later}, s.read(time.Time{}, 1))
}

func (s *testSuite) TestNoLimitIsTheDefaultAndNoLimitPassesTheMost() {
	for n := range reports_query.MaxLimit + 5 {
		_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `
			INSERT INTO reports (id, reporter_account_id, reported_account_id, reason, text, message_id, created_at)
			VALUES ($1, $2, $3, 'other', '', '', $4)
		`, uuid.New(), uuid.New(), uuid.UUID(bob), start.Add(time.Duration(n)*time.Second))
		s.Require().NoError(err)
	}

	s.Len(s.read(time.Time{}, 0), reports_query.DefaultLimit)
	s.Len(s.read(time.Time{}, reports_query.MaxLimit+5), reports_query.MaxLimit)
}

func (s *testSuite) TestAReasonTheDatabaseNamesAndTheWireCannotCarryRefusesTheRead() {
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `ALTER TYPE report_reason ADD VALUE IF NOT EXISTS 'boring'`)
	s.Require().NoError(err)
	_, err = s.db.On(s.T().Context()).ExecContext(s.T().Context(), `
		INSERT INTO reports (id, reporter_account_id, reported_account_id, reason, text, message_id, created_at)
		VALUES ($1, $2, $3, 'boring', '', '', $4)
	`, uuid.New(), uuid.UUID(ada), uuid.UUID(bob), start)
	s.Require().NoError(err)

	_, err = s.query.Reports(s.T().Context(), time.Time{}, 0, false)

	s.Require().ErrorIs(err, reports_query.ErrUnknownReason)
}

func (s *testSuite) TestReportsNobodyCouldNameAreRefused() {
	s.reported(1, ada, bob, reports.Complaint{Reason: reports.ReasonOther}, start)
	cause := errors.New("the player module is down")
	s.names.err = cause

	_, err := s.query.Reports(s.T().Context(), time.Time{}, 0, false)

	s.Require().ErrorIs(err, cause)
}

func (s *testSuite) TestAFailedReadIsAnError() {
	ctx, cancel := context.WithCancel(s.T().Context())
	cancel()

	_, err := s.query.Reports(ctx, time.Time{}, 0, false)

	s.Error(err, "not an empty list")
}

func (s *testSuite) decided(n byte, outcome decisions.Outcome, comment string, at time.Time) {
	decision, err := decisions.NewDecision(reports.ReportID{15: n}, outcome, comment, at)
	s.Require().NoError(err)
	s.Require().NoError(postgres_decision_store.New(s.db).Save(s.T().Context(), decision))
}

func (s *testSuite) TestADecidedReportCarriesItsDecision() {
	report := s.reported(1, ada, bob, reports.Complaint{Reason: reports.ReasonChatAbuse}, start)
	report.Reason = supportv1.ReportReason_REPORT_REASON_CHAT_ABUSE
	s.decided(1, decisions.OutcomeActed, "muted for a day", start.Add(time.Hour))
	report.Decision = &supportv1.ReportDecision{
		Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_ACTED, Comment: "muted for a day",
		DecidedAt: timestamppb.New(start.Add(time.Hour)),
	}

	s.same([]*supportv1.Report{report}, s.read(time.Time{}, 0))
}

func (s *testSuite) TestTheOpenReportsLeaveOutTheDecidedOnes() {
	s.reported(1, ada, bob, reports.Complaint{Reason: reports.ReasonOther}, start)
	open := s.reported(2, bob, ada, reports.Complaint{Reason: reports.ReasonOther}, start.Add(time.Minute))
	open.Reason = supportv1.ReportReason_REPORT_REASON_OTHER
	s.decided(1, decisions.OutcomeDismissed, "", start.Add(time.Hour))

	answer, err := s.query.Reports(s.T().Context(), time.Time{}, 0, true)
	s.Require().NoError(err)

	s.same([]*supportv1.Report{open}, answer.GetReports())
	s.Len(s.read(time.Time{}, 0), 2, "every report unless asked")
}

func (s *testSuite) TestEveryOutcomeTheDomainKnowsIsRead() {
	for n, outcome := range []decisions.Outcome{decisions.OutcomeActed, decisions.OutcomeDismissed} {
		s.reported(byte(n+1), ada, bob, reports.Complaint{Reason: reports.ReasonOther}, start.Add(time.Duration(n)*24*time.Hour))
		s.decided(byte(n+1), outcome, "", start.Add(time.Duration(n)*24*time.Hour+time.Hour))
	}

	got := s.read(time.Time{}, 0)

	s.Require().Len(got, 2)
	s.Equal([]supportv1.ReportOutcome{
		supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED, supportv1.ReportOutcome_REPORT_OUTCOME_ACTED,
	}, []supportv1.ReportOutcome{got[0].GetDecision().GetOutcome(), got[1].GetDecision().GetOutcome()})
}

func (s *testSuite) TestAnOutcomeTheDatabaseNamesAndTheWireCannotCarryRefusesTheRead() {
	s.reported(1, ada, bob, reports.Complaint{Reason: reports.ReasonOther}, start)
	_, err := s.db.On(s.T().Context()).ExecContext(s.T().Context(), `ALTER TYPE report_outcome ADD VALUE IF NOT EXISTS 'escalated'`)
	s.Require().NoError(err)
	_, err = s.db.On(s.T().Context()).ExecContext(s.T().Context(), `
		INSERT INTO report_decisions (report_id, outcome, comment, decided_at) VALUES ($1, 'escalated', '', $2)
	`, uuid.UUID(reports.ReportID{15: 1}), start)
	s.Require().NoError(err)

	_, err = s.query.Reports(s.T().Context(), time.Time{}, 0, false)

	s.Require().ErrorIs(err, reports_query.ErrUnknownOutcome)
}
