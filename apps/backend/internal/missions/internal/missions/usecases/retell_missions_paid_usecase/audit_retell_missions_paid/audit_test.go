package audit_retell_missions_paid_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_paid_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_paid_usecase/audit_retell_missions_paid"
)

type stubUseCase struct {
	out retell_missions_paid_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, retell_missions_paid_usecase.In) (retell_missions_paid_usecase.Out, error) {
	return s.out, s.err
}

func TestARetellIsLoggedWithTheAccountsItTold(t *testing.T) {
	var logs bytes.Buffer
	audited := audit_retell_missions_paid.New(stubUseCase{out: retell_missions_paid_usecase.Out{Accounts: 812}},
		slog.New(slog.NewTextHandler(&logs, nil)))

	out, err := audited.Execute(t.Context(), retell_missions_paid_usecase.In{DryRun: true})

	require.NoError(t, err)
	assert.Equal(t, 812, out.Accounts)
	assert.Contains(t, logs.String(), `level=WARN msg="admin missions paid retold" dry_run=true accounts=812`)
}

func TestARetellCutShortIsLoggedWithWhatItToldBefore(t *testing.T) {
	var logs bytes.Buffer
	audited := audit_retell_missions_paid.New(stubUseCase{out: retell_missions_paid_usecase.Out{Accounts: 500}, err: errors.New("down")},
		slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := audited.Execute(t.Context(), retell_missions_paid_usecase.In{})

	require.Error(t, err)
	assert.Contains(t, logs.String(), `level=WARN msg="admin missions paid retold in part" dry_run=false accounts=500 error=down`)
}
