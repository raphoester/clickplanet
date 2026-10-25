package fill_charges_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/fills/usecases/start_fill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/fill_charges_handler"
)

type stubUseCase struct {
	err   error
	asked []start_fill_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in start_fill_usecase.In) error {
	s.asked = append(s.asked, in)
	return s.err
}

func fill(t *testing.T, handler fill_charges_handler.FillChargesHandler) error {
	t.Helper()

	_, err := handler.FillCharges(t.Context(), connect.NewRequest(&planetv1.FillChargesRequest{
		RequestId: "season-0:finale:fill",
	}))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestTheFillIsStartedAsTheHandlersSource(t *testing.T) {
	for _, source := range []fills.Source{fills.SourceInternal, fills.SourceOperator} {
		useCase := &stubUseCase{}

		require.NoError(t, fill(t, fill_charges_handler.New(useCase, source)))

		assert.Equal(t, []start_fill_usecase.In{{Request: "season-0:finale:fill", Source: source}}, useCase.asked)
	}
}

func TestARequestThatCannotStartAFillIsAnInvalidArgument(t *testing.T) {
	for _, refusal := range []error{fills.ErrNoRequest, fills.ErrLongRequest} {
		err := fill(t, fill_charges_handler.New(&stubUseCase{err: refusal}, fills.SourceOperator))

		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", refusal)
	}
}

func TestAFailureIsPassedOnForTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	err := fill(t, fill_charges_handler.New(&stubUseCase{err: down}, fills.SourceInternal))

	require.ErrorIs(t, err, down)
}
