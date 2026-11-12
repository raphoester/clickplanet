package decide_report_usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

var now = time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

type recordedDecisions struct {
	saved []decisions.Snapshot
	err   error
}

func (r *recordedDecisions) Save(_ context.Context, decision decisions.Decision) error {
	if r.err != nil {
		return r.err
	}
	r.saved = append(r.saved, decision.TakeSnapshot())
	return nil
}

func (r *recordedDecisions) SaveOnce(ctx context.Context, decision decisions.Decision) (bool, error) {
	for _, saved := range r.saved {
		if saved.Report == decision.TakeSnapshot().Report {
			return false, r.err
		}
	}
	return r.err == nil, r.Save(ctx, decision)
}

func decide(t *testing.T, saver *recordedDecisions, in decide_report_usecase.In) (time.Time, error) {
	t.Helper()

	return decide_report_usecase.New(saver, cptime.NewFixedClock(now)).Execute(t.Context(), in) //nolint:wrapcheck // the tests read the sentinel.
}

func TestADecisionIsKeptAtTheTimeItWasMade(t *testing.T) {
	saver := &recordedDecisions{}

	decidedAt, err := decide(t, saver, decide_report_usecase.In{
		Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "muted",
	})
	require.NoError(t, err)

	assert.Equal(t, now, decidedAt)
	assert.Equal(t, []decisions.Snapshot{
		{Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "muted", At: now},
	}, saver.saved)
}

func TestADecisionThatBreaksARuleIsNotKept(t *testing.T) {
	saver := &recordedDecisions{}

	_, err := decide(t, saver, decide_report_usecase.In{Report: reports.ReportID{15: 1}, Outcome: "banned"})

	require.ErrorIs(t, err, decisions.ErrUnknownOutcome)
	assert.Empty(t, saver.saved)
}

func TestADecisionOnceIsKeptForAReportNotDecidedYet(t *testing.T) {
	saver := &recordedDecisions{}

	decidedAt, err := decide(t, saver, decide_report_usecase.In{
		Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "Claude judge: removed", Once: true,
	})

	require.NoError(t, err)
	assert.Equal(t, now, decidedAt)
	assert.Equal(t, []decisions.Snapshot{
		{Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "Claude judge: removed", At: now},
	}, saver.saved)
}

func TestADecisionOnceLeavesAReportDecidedAlready(t *testing.T) {
	saver := &recordedDecisions{}
	_, err := decide(t, saver, decide_report_usecase.In{
		Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeDismissed, Comment: "the operator's",
	})
	require.NoError(t, err)

	_, err = decide(t, saver, decide_report_usecase.In{
		Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "the judge's", Once: true,
	})

	require.ErrorIs(t, err, decisions.ErrDecidedAlready)
	assert.Equal(t, "the operator's", saver.saved[0].Comment)
	assert.Len(t, saver.saved, 1)
}

func TestTheOperatorsDecisionReplacesOneMadeOnce(t *testing.T) {
	saver := &recordedDecisions{}
	_, err := decide(t, saver, decide_report_usecase.In{
		Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "the judge's", Once: true,
	})
	require.NoError(t, err)

	_, err = decide(t, saver, decide_report_usecase.In{
		Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeDismissed, Comment: "the operator's",
	})

	require.NoError(t, err)
	assert.Len(t, saver.saved, 2, "saved again: the store keeps the last")
}
