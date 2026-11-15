package retell_missions_paid_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_paid_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/retell_missions_paid_handler"
)

type stubUseCase struct {
	out   retell_missions_paid_usecase.Out
	err   error
	asked *retell_missions_paid_usecase.In
}

func (s stubUseCase) Execute(_ context.Context, in retell_missions_paid_usecase.In) (retell_missions_paid_usecase.Out, error) {
	*s.asked = in
	return s.out, s.err
}

func retell(t *testing.T, useCase stubUseCase, dryRun bool) (*connect.Response[missionsv1.RetellMissionsPaidResponse], error) {
	t.Helper()

	return retell_missions_paid_handler.New(useCase).RetellMissionsPaid(t.Context(), //nolint:wrapcheck // the tests read the error.
		connect.NewRequest(&missionsv1.RetellMissionsPaidRequest{DryRun: dryRun}))
}

func TestTheAnswerSaysHowManyAccountsWereTold(t *testing.T) {
	asked := &retell_missions_paid_usecase.In{}

	res, err := retell(t, stubUseCase{out: retell_missions_paid_usecase.Out{Accounts: 1_204}, asked: asked}, true)

	require.NoError(t, err)
	assert.Equal(t, uint32(1_204), res.Msg.GetAccounts())
	assert.True(t, asked.DryRun)
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	down := errors.New("down")

	_, err := retell(t, stubUseCase{err: down, asked: &retell_missions_paid_usecase.In{}}, false)

	require.ErrorIs(t, err, down)
}
