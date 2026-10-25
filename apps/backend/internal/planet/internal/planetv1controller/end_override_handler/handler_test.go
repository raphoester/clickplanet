package end_override_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/end_override_handler"
)

type stubUseCase struct {
	err   error
	calls int
}

func (s *stubUseCase) Execute(context.Context) error {
	s.calls++
	return s.err
}

func TestEndingAsksTheUseCase(t *testing.T) {
	useCase := &stubUseCase{}

	_, err := end_override_handler.New(useCase).EndOverride(t.Context(), connect.NewRequest(&planetv1.EndOverrideRequest{}))

	require.NoError(t, err)
	assert.Equal(t, 1, useCase.calls)
}

func TestAFailureIsPassedOnForTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := end_override_handler.New(&stubUseCase{err: down}).
		EndOverride(t.Context(), connect.NewRequest(&planetv1.EndOverrideRequest{}))

	require.ErrorIs(t, err, down)
}
