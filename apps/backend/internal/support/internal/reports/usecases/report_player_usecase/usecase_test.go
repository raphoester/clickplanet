package report_player_usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/inmemory_report_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports/usecases/report_player_usecase"
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(testSuite))
}

var (
	now      = time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)
	ada      = reports.AccountID{15: 1}
	bob      = reports.AccountID{15: 2}
	cheating = reports.Complaint{Reason: reports.ReasonCheating}
)

type failingIDs struct{}

func (failingIDs) NewID() (reports.ReportID, error) {
	return reports.ReportID{}, errors.New("no entropy")
}

type testSuite struct {
	suite.Suite

	clock   *cptime.FixedClock
	reports *inmemory_report_storage.Storage
	useCase *report_player_usecase.UseCase
}

func (s *testSuite) SetupTest() {
	s.clock = cptime.NewFixedClock(now)
	s.reports = inmemory_report_storage.New()
	s.useCase = report_player_usecase.New(&reports.SequentialIDs{}, s.clock, s.reports)
}

func (s *testSuite) report(in report_player_usecase.In) error {
	return s.useCase.Execute(s.T().Context(), in)
}

func (s *testSuite) TestAReportIsKeptAgainstTheAccountItNames() {
	err := s.report(report_player_usecase.In{
		Reporter: ada, Reported: bob,
		Complaint: reports.Complaint{Reason: reports.ReasonChatAbuse, Text: "  it insults everyone  ", Message: "message-1"},
	})
	s.Require().NoError(err)

	s.Equal([]reports.Snapshot{{
		ID: reports.ReportID{15: 1}, Reporter: ada, Reported: bob, At: now,
		Complaint: reports.Complaint{Reason: reports.ReasonChatAbuse, Text: "it insults everyone", Message: "message-1"},
	}}, s.reports.Kept())
}

func (s *testSuite) TestAReportThatNamesNobodyIsRefused() {
	err := s.report(report_player_usecase.In{Reporter: ada, Complaint: cheating})

	s.Require().ErrorIs(err, reports.ErrInvalidReport)
	s.Empty(s.reports.Kept())
}

func (s *testSuite) TestAReasonNobodyKnowsIsRefusedAndNothingIsKept() {
	err := s.report(report_player_usecase.In{Reporter: ada, Reported: bob, Complaint: reports.Complaint{Reason: "boring"}})

	s.Require().ErrorIs(err, reports.ErrUnknownReason)
	s.Empty(s.reports.Kept())
}

func (s *testSuite) TestATextTooLongIsRefusedAndNothingIsKept() {
	err := s.report(report_player_usecase.In{
		Reporter: ada, Reported: bob,
		Complaint: reports.Complaint{Reason: reports.ReasonOther, Text: strings.Repeat("a", notes.MaxTextLength+1)},
	})

	s.Require().ErrorIs(err, notes.ErrInvalidText)
	s.Empty(s.reports.Kept())
}

func (s *testSuite) TestAPlayerCannotReportItself() {
	err := s.report(report_player_usecase.In{Reporter: ada, Reported: ada, Complaint: cheating})

	s.Require().ErrorIs(err, reports.ErrOwnAccount)
	s.Empty(s.reports.Kept())
}

func (s *testSuite) TestASecondReportOfTheSamePlayerTheSameDayIsRefusedAndTheNextDayIsKept() {
	in := report_player_usecase.In{Reporter: ada, Reported: bob, Complaint: cheating}
	s.Require().NoError(s.report(in))

	s.Require().ErrorIs(s.report(in), reports.ErrReportedAlready)
	s.Len(s.reports.Kept(), 1)

	s.clock.Advance(4 * time.Hour)
	s.Require().NoError(s.report(in), "past midnight UTC")
	s.Len(s.reports.Kept(), 2)
}

func (s *testSuite) TestWhatOthersFiledAndWhatIsFiledAboutOthersRefusesNothing() {
	eve := reports.AccountID{15: 3}
	s.reports.Seed(
		reports.Snapshot{ID: reports.ReportID{14: 1}, Reporter: eve, Reported: bob, Complaint: cheating, At: now},
		reports.Snapshot{ID: reports.ReportID{14: 2}, Reporter: ada, Reported: eve, Complaint: cheating, At: now},
		reports.Snapshot{ID: reports.ReportID{14: 3}, Reporter: bob, Reported: ada, Complaint: cheating, At: now},
	)

	s.Require().NoError(s.report(report_player_usecase.In{Reporter: ada, Reported: bob, Complaint: cheating}))

	s.Len(s.reports.Kept(), 4)
}

func (s *testSuite) TestAReportWithNoIDIsNotKept() {
	useCase := report_player_usecase.New(failingIDs{}, s.clock, s.reports)

	err := useCase.Execute(s.T().Context(), report_player_usecase.In{Reporter: ada, Reported: bob, Complaint: cheating})

	s.Require().Error(err)
	s.Empty(s.reports.Kept())
}

func (s *testSuite) TestAFailureToReadWhatWasFiledFailsAndKeepsNothing() {
	cause := errors.New("postgres is down")
	s.reports.FailWith(cause)

	err := s.report(report_player_usecase.In{Reporter: ada, Reported: bob, Complaint: cheating})

	s.Require().ErrorIs(err, cause)
	s.Empty(s.reports.Kept())
}

type readOnly struct {
	*inmemory_report_storage.Storage

	refused error
}

func (r readOnly) Save(context.Context, ...reports.Report) error {
	return r.refused
}

func (s *testSuite) TestAReportThatCannotBeKeptFails() {
	cause := errors.New("postgres is down")
	useCase := report_player_usecase.New(&reports.SequentialIDs{}, s.clock, readOnly{Storage: s.reports, refused: cause})

	err := useCase.Execute(s.T().Context(), report_player_usecase.In{Reporter: ada, Reported: bob, Complaint: cheating})

	s.Require().ErrorIs(err, cause)
}
