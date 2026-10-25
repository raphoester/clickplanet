package audit_end_override_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/end_override_usecase/audit_end_override"
)

type stubUseCase struct{ err error }

func (s stubUseCase) Execute(context.Context) error { return s.err }

func TestEveryEndIsLogged(t *testing.T) {
	var logs bytes.Buffer

	require.NoError(t, audit_end_override.New(stubUseCase{}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context()))

	assert.Contains(t, logs.String(), `level=WARN msg="admin rule override ended"`)
}

func TestARefusedEndIsLoggedWithWhyAndPassedOn(t *testing.T) {
	var logs bytes.Buffer
	cause := errors.New("postgres is down")

	err := audit_end_override.New(stubUseCase{err: cause}, slog.New(slog.NewTextHandler(&logs, nil))).Execute(t.Context())

	require.ErrorIs(t, err, cause)
	assert.Contains(t, logs.String(), `level=WARN msg="admin rule override end refused" error="postgres is down"`)
}
