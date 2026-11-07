package decide_report_handler_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/commands/decide_report_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

var report = reports.ReportID{15: 1}

type recording struct {
	err error
	in  []decide_report_usecase.In
}

func (r *recording) Execute(_ context.Context, in decide_report_usecase.In) (time.Time, error) {
	r.in = append(r.in, in)
	return time.Time{}, r.err
}

func handle(useCase *recording, command *supportv1.DecideReport) error {
	return decide_report_handler.New(useCase, slog.New(slog.DiscardHandler)).Handle(context.Background(), command) //nolint:wrapcheck // the tests read the sentinel.
}

func TestACommandDecidesItsReportOnce(t *testing.T) {
	useCase := &recording{}

	err := handle(useCase, &supportv1.DecideReport{
		ReportId: uuid.UUID(report).String(), Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_ACTED,
		Comment: "Claude judge: offence 1 (chat): a slur",
	})

	require.NoError(t, err)
	assert.Equal(t, []decide_report_usecase.In{{
		Report: report, Outcome: decisions.OutcomeActed, Comment: "Claude judge: offence 1 (chat): a slur", Once: true,
	}}, useCase.in)
}

func TestAReportDecidedAlreadyOrNoLongerKeptIsDoneWith(t *testing.T) {
	for _, cause := range []error{decisions.ErrDecidedAlready, decisions.ErrUnknownReport} {
		err := handle(&recording{err: cause}, &supportv1.DecideReport{
			ReportId: uuid.UUID(report).String(), Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED,
		})

		require.NoError(t, err, "a try again would change nothing: %v", cause)
	}
}

func TestAFailureIsTriedAgain(t *testing.T) {
	cause := errors.New("postgres is down")

	err := handle(&recording{err: cause}, &supportv1.DecideReport{
		ReportId: uuid.UUID(report).String(), Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED,
	})

	require.ErrorIs(t, err, cause)
}

func TestACommandThatNamesNoReportOrNoOutcomeIsRefused(t *testing.T) {
	useCase := &recording{}

	require.ErrorIs(t, handle(useCase, &supportv1.DecideReport{
		ReportId: "nope", Outcome: supportv1.ReportOutcome_REPORT_OUTCOME_ACTED,
	}), reports.ErrInvalidReportID)
	require.ErrorIs(t, handle(useCase, &supportv1.DecideReport{ReportId: uuid.UUID(report).String()}),
		decide_report_handler.ErrUnknownOutcome)
	assert.Empty(t, useCase.in)
}
