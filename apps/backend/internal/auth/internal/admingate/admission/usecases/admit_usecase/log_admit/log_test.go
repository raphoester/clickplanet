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
	operator admission.Operator
	err      error
}

func (s stubUseCase) Execute(context.Context, admission.Visit) (admission.Operator, error) {
	return s.operator, s.err
}

func TestARefusedCallIsLoggedWithItsOriginAndWithoutItsCookie(t *testing.T) {
	var logs bytes.Buffer
	cause := fmt.Errorf("%w: %q", admission.ErrForeignOrigin, "https://clickplanet.lol")
	useCase := log_admit.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil)))

	_, err := useCase.Execute(t.Context(), admission.Visit{Origin: "https://clickplanet.lol", Cookie: "cp_sid=the-secret"})

	require.ErrorIs(t, err, admission.ErrForeignOrigin)
	assert.Contains(t, logs.String(), `level=WARN msg="admin call refused" origin=https://clickplanet.lol`)
	assert.NotContains(t, logs.String(), "the-secret")
}

func TestAnAdmittedCallIsNotLoggedHere(t *testing.T) {
	var logs bytes.Buffer
	useCase := log_admit.New(stubUseCase{operator: "the operator"}, slog.New(slog.NewTextHandler(&logs, nil)))

	operator, err := useCase.Execute(t.Context(), admission.Visit{Origin: "https://admin.clickplanet.lol"})

	require.NoError(t, err)
	assert.Equal(t, admission.Operator("the operator"), operator)
	assert.Empty(t, logs.String(), "the call log names it, once the call is done")
}
