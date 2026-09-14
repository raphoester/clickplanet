package audit_reassign_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/reassign_country_usecase/audit_reassign"
)

type stubUseCase struct {
	out reassign_country_usecase.Out
	err error
}

func (s stubUseCase) Execute(context.Context, reassign_country_usecase.In) (reassign_country_usecase.Out, error) {
	return s.out, s.err
}

func audited(inner stubUseCase) (*audit_reassign.Audited, *bytes.Buffer) {
	var logs bytes.Buffer
	return audit_reassign.New(inner, slog.New(slog.NewTextHandler(&logs, nil))), &logs
}

func TestAReassignmentIsLoggedWithItsCounts(t *testing.T) {
	want := reassign_country_usecase.Out{FromBefore: 22040, Moved: 22040, ToAfter: 22040}
	useCase, logs := audited(stubUseCase{out: want})

	out, err := useCase.Execute(t.Context(), reassign_country_usecase.In{From: "dz", To: "fr"})
	require.NoError(t, err)

	assert.Equal(t, want, out)
	assert.Contains(t, logs.String(), `level=WARN msg="admin country reassignment" from=dz to=fr dryRun=false fromBefore=22040`)
	assert.Contains(t, logs.String(), "moved=22040")
}

func TestADryRunIsLoggedToo(t *testing.T) {
	useCase, logs := audited(stubUseCase{})

	_, err := useCase.Execute(t.Context(), reassign_country_usecase.In{From: "dz", To: "fr", DryRun: true})
	require.NoError(t, err)

	assert.Contains(t, logs.String(), "dryRun=true")
}

func TestAFailureIsLoggedWithHowFarItGot(t *testing.T) {
	cause := errors.New("reassignment interrupted")
	useCase, logs := audited(stubUseCase{out: reassign_country_usecase.Out{Moved: 512}, err: cause})

	out, err := useCase.Execute(t.Context(), reassign_country_usecase.In{From: "dz", To: "fr"})

	require.ErrorIs(t, err, cause)
	assert.Equal(t, 512, out.Moved)
	assert.Contains(t, logs.String(), `msg="admin country reassignment failed"`)
	assert.Contains(t, logs.String(), "moved=512")
	assert.Contains(t, logs.String(), `error="reassignment interrupted"`)
}
