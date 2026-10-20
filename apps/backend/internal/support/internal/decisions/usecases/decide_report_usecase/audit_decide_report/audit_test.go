package audit_decide_report_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase/audit_decide_report"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type stubUseCase struct {
	err error
}

func (s stubUseCase) Execute(context.Context, decide_report_usecase.In) (time.Time, error) {
	return time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC), s.err
}

func run(t *testing.T, inner stubUseCase) (string, error) {
	t.Helper()

	var logs bytes.Buffer
	_, err := audit_decide_report.New(inner, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context(),
		decide_report_usecase.In{Report: reports.ReportID{15: 1}, Outcome: decisions.OutcomeActed, Comment: "muted"})
	return logs.String(), err
}

func TestEveryDecisionIsLogged(t *testing.T) {
	logs, err := run(t, stubUseCase{})

	require.NoError(t, err)
	assert.Contains(t, logs,
		`level=WARN msg="admin report decision" report=00000000-0000-0000-0000-000000000001 outcome=acted comment=muted`)
}

func TestARefusedDecisionIsLoggedToo(t *testing.T) {
	cause := errors.New("postgres is down")

	logs, err := run(t, stubUseCase{err: cause})

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs, `level=WARN msg="admin report decision failed" report=00000000-0000-0000-0000-000000000001`)
}
