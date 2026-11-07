package log_judgement_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/usecases/judge_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/judgements/usecases/judge_report_usecase/log_judgement"
)

var report = judgements.Report{ID: judgements.ReportID{15: 1}, Message: "m-2"}

type stub struct {
	out judge_report_usecase.Out
	err error
}

func (s stub) Execute(context.Context, judgements.Report) (judge_report_usecase.Out, error) {
	return s.out, s.err
}

func logged(t *testing.T, verdict judgements.Verdict, flagged bool, spend judgements.Spend) string {
	t.Helper()
	judgement, err := judgements.Snapshot{
		Report: report.ID, Verdict: verdict, Flagged: flagged, Reason: "child_safety: an adult asks", At: time.Now(),
	}.Restore()
	require.NoError(t, err)
	var out bytes.Buffer
	_, err = log_judgement.New(stub{out: judge_report_usecase.Out{Judgement: judgement, Spend: spend}},
		slog.New(slog.NewTextHandler(&out, nil))).Execute(t.Context(), report)
	require.NoError(t, err)
	return out.String()
}

func TestWhatTheOperatorShouldSeeIsAWarning(t *testing.T) {
	line := logged(t, judgements.LeftOpen, true, judgements.Spend{})

	assert.Contains(t, line, "level=WARN")
	assert.Contains(t, line, "chat report left to the operator")
	assert.Contains(t, line, "child_safety")
	assert.Contains(t, line, report.ID.String())
}

func TestTheRestIsInformationWithWhatItCost(t *testing.T) {
	line := logged(t, judgements.Dismissed, false, judgements.Spend{Model: "claude-haiku-5-5", Input: 3000, Output: 300})

	assert.Contains(t, line, "level=INFO")
	assert.Contains(t, line, "inputTokens=3000")
	assert.Contains(t, line, "dollars=")
}

func TestAFailureIsPassedOnUnlogged(t *testing.T) {
	var out bytes.Buffer
	cause := errors.New("overloaded")

	_, err := log_judgement.New(stub{err: cause}, slog.New(slog.NewTextHandler(&out, nil))).Execute(t.Context(), report)

	require.ErrorIs(t, err, cause)
	assert.Empty(t, out.String(), "the subscriber's log says it")
}
