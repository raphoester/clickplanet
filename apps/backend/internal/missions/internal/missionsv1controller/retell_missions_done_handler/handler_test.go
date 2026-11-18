package retell_missions_done_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_done_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missionsv1controller/retell_missions_done_handler"
)

type stubUseCase struct {
	out   retell_missions_done_usecase.Out
	asked *retell_missions_done_usecase.In
}

func (s stubUseCase) Execute(_ context.Context, in retell_missions_done_usecase.In) (retell_missions_done_usecase.Out, error) {
	*s.asked = in
	return s.out, nil
}

func retell(t *testing.T, useCase stubUseCase, dryRun bool) (*connect.Response[missionsv1.RetellMissionsDoneResponse], error) {
	t.Helper()

	return retell_missions_done_handler.New(useCase).RetellMissionsDone(t.Context(), //nolint:wrapcheck // the tests read the error.
		connect.NewRequest(&missionsv1.RetellMissionsDoneRequest{DryRun: dryRun}))
}

func TestTheAnswerSaysHowManyAccountsWereTold(t *testing.T) {
	asked := &retell_missions_done_usecase.In{}

	res, err := retell(t, stubUseCase{out: retell_missions_done_usecase.Out{Accounts: 1_204}, asked: asked}, true)

	require.NoError(t, err)
	assert.Equal(t, uint32(1_204), res.Msg.GetAccounts())
	assert.True(t, asked.DryRun)
}
