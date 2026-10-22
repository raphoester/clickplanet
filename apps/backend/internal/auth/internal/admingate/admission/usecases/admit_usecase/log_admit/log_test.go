package log_admit_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission/usecases/admit_usecase/log_admit"
)

type stubUseCase struct {
	caller admission.Caller
	err    error
}

func (s stubUseCase) Execute(context.Context, admission.Visit) (admission.Caller, error) {
	return s.caller, s.err
}

func TestARefusedCallIsLoggedWithItsOriginAndWithoutItsCookie(t *testing.T) {
	var logs bytes.Buffer
	cause := fmt.Errorf("%w: %q", admission.ErrForeignOrigin, "https://clickplanet.lol")
	useCase := log_admit.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := useCase.Execute(t.Context(), admission.Visit{Origin: "https://clickplanet.lol", Cookie: "cp_sid=the-secret"})

	require.ErrorIs(t, err, admission.ErrForeignOrigin)
	assert.Contains(t, logs.String(), `level=WARN msg="admin call refused" origin=https://clickplanet.lol token=false`)
	assert.NotContains(t, logs.String(), "the-secret")
}

func TestARefusedTokenIsLoggedWithoutTheToken(t *testing.T) {
	var logs bytes.Buffer
	cause := fmt.Errorf("%w: nobody holds this token", admission.ErrNoToken)
	useCase := log_admit.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := useCase.Execute(t.Context(), admission.Visit{Token: "cpat_the-secret"})

	require.ErrorIs(t, err, admission.ErrNoToken)
	assert.Contains(t, logs.String(), `msg="admin call refused" origin="" token=true`)
	assert.NotContains(t, logs.String(), "the-secret")
}

func TestAnAdmittedCallIsNotLoggedHere(t *testing.T) {
	var logs bytes.Buffer
	useCase := log_admit.New(stubUseCase{caller: admission.SessionOf("the operator")}, slog.New(slog.NewTextHandler(&logs, nil)))

	caller, err := useCase.Execute(t.Context(), admission.Visit{Origin: "https://backoffice.clickplanet.lol"})

	require.NoError(t, err)
	assert.Equal(t, admission.Operator("the operator"), caller.Operator())
	assert.Empty(t, logs.String(), "the call log names it, once the call is done")
}
