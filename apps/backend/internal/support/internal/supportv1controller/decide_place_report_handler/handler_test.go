package decide_place_report_handler_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/notes"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/supportv1controller/decide_place_report_handler"
)

type stubUseCase struct {
	in        *decide_report_usecase.In[places.ReportID]
	decidedAt time.Time
	err       error
}

func (s *stubUseCase) Execute(_ context.Context, in decide_report_usecase.In[places.ReportID]) (time.Time, error) {
	s.in = &in
	return s.decidedAt, s.err
}

const reportID = "00000000-0000-0000-0000-000000000001"

func decide(t *testing.T, useCase *stubUseCase, req *supportv1.DecidePlaceReportRequest) (*supportv1.DecidePlaceReportResponse, error) {
	t.Helper()

	res, err := decide_place_report_handler.New(useCase).DecidePlaceReport(t.Context(), connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("decision refused: %w", err)
	}
	return res.Msg, nil
}

func TestTheDecisionReachesTheUseCaseAndWhenComesBack(t *testing.T) {
	at := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	useCase := &stubUseCase{decidedAt: at}

	res, err := decide(t, useCase, &supportv1.DecidePlaceReportRequest{
		ReportId: reportID, Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED, Comment: "a duel",
	})
	require.NoError(t, err)

	assert.Equal(t, &decide_report_usecase.In[places.ReportID]{
		Report: places.ReportID{15: 1}, Outcome: decisions.OutcomeDismissed, Comment: "a duel",
	}, useCase.in)
	assert.Equal(t, at, res.GetDecidedAt().AsTime())
}

func TestAnIDThatIsNotAReportIsRefusedBeforeTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := decide(t, useCase, &supportv1.DecidePlaceReportRequest{
		ReportId: "not-a-report", Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_ACTED,
	})

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Nil(t, useCase.in)
}

func TestErrorsMapToTheirCodes(t *testing.T) {
	for err, code := range map[error]connect.Code{
		fmt.Errorf("refused: %w", decisions.ErrUnknownOutcome): connect.CodeInvalidArgument,
		fmt.Errorf("refused: %w", notes.ErrInvalidText):        connect.CodeInvalidArgument,
		fmt.Errorf("refused: %w", decisions.ErrUnknownReport):  connect.CodeNotFound,
		assert.AnError: connect.CodeUnknown,
	} {
		_, got := decide(t, &stubUseCase{err: err}, &supportv1.DecidePlaceReportRequest{ReportId: reportID})
		assert.Equal(t, code, connect.CodeOf(got), err.Error())
	}
}
