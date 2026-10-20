package decide_report_usecase_test

import (
	"context"
	"errors"
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
	saved []decisions.Snapshot[reports.ReportID]
	err   error
}

func (r *recordedDecisions) Save(_ context.Context, decision decisions.Decision[reports.ReportID]) error {
	if r.err != nil {
		return r.err
	}
	r.saved = append(r.saved, decision.TakeSnapshot())
	return nil
}

func decide(t *testing.T, saver *recordedDecisions, in decide_report_usecase.In[reports.ReportID]) (time.Time, error) {
	t.Helper()

	return decide_report_usecase.New[reports.ReportID](saver, cptime.NewFixedClock(now)).Execute(t.Context(), in) //nolint:wrapcheck // the tests read the sentinel.
}

func TestADecisionIsKeptAtTheTimeItWasMade(t *testing.T) {
	saver := &recordedDecisions{}

	decidedAt, err := decide(t, saver, decide_report_usecase.In[reports.ReportID]{
		Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "muted",
	})
	require.NoError(t, err)

	assert.Equal(t, now, decidedAt)
	assert.Equal(t, []decisions.Snapshot[reports.ReportID]{
		{Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "muted", At: now},
	}, saver.saved)
}

func TestADecisionThatBreaksARuleIsNotKept(t *testing.T) {
	saver := &recordedDecisions{}

	_, err := decide(t, saver, decide_report_usecase.In[reports.ReportID]{Report: reports.ReportID{15: 1}, Outcome: "banned"})

	require.ErrorIs(t, err, decisions.ErrUnknownOutcome)
	assert.Empty(t, saver.saved)
}

func TestADecisionThatCannotBeKeptFails(t *testing.T) {
	cause := errors.New("postgres is down")

	_, err := decide(t, &recordedDecisions{err: cause}, decide_report_usecase.In[reports.ReportID]{
		Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeDismissed,
	})

	require.ErrorIs(t, err, cause)
}
