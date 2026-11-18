package reconcile_levels_handler_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels/usecases/reconcile_levels_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/reconcile_levels_handler"
)

type stubUseCase struct {
	reconciled reconcile_levels_usecase.Reconciled
	dryRuns    *[]bool
}

func (s stubUseCase) Execute(_ context.Context, dryRun bool) (reconcile_levels_usecase.Reconciled, error) {
	*s.dryRuns = append(*s.dryRuns, dryRun)
	return s.reconciled, nil
}

func reconcile(t *testing.T, useCase stubUseCase, dryRun bool) (*connect.Response[progressionv1.ReconcileLevelsResponse], error) {
	t.Helper()

	return reconcile_levels_handler.New(useCase).ReconcileLevels(t.Context(), //nolint:wrapcheck // the tests read the error.
		connect.NewRequest(&progressionv1.ReconcileLevelsRequest{DryRun: dryRun}))
}

func TestTheAnswerSaysHowManyLevelsWentUpDownAndAway(t *testing.T) {
	dryRuns := []bool{}
	useCase := stubUseCase{reconciled: reconcile_levels_usecase.Reconciled{Raised: 2_700, Removed: 3}, dryRuns: &dryRuns}

	res, err := reconcile(t, useCase, true)

	require.NoError(t, err)
	assert.Equal(t, uint32(2_700), res.Msg.GetRaised())
	assert.Equal(t, uint32(3), res.Msg.GetRemoved())

	_, err = reconcile(t, useCase, false)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false}, dryRuns, "the dry run is the caller's")
}
