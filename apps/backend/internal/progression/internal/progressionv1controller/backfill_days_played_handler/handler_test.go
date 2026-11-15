package backfill_days_played_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	progressionv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/progression/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/progressionv1controller/backfill_days_played_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/backfill_days_played_usecase"
)

type stubUseCase struct {
	backfilled backfill_days_played_usecase.Backfilled
	err        error
	dryRuns    *[]bool
}

func (s stubUseCase) Execute(_ context.Context, dryRun bool) (backfill_days_played_usecase.Backfilled, error) {
	*s.dryRuns = append(*s.dryRuns, dryRun)
	return s.backfilled, s.err
}

func backfill(t *testing.T, useCase stubUseCase, dryRun bool) (*connect.Response[progressionv1.BackfillDaysPlayedResponse], error) {
	t.Helper()

	return backfill_days_played_handler.New(useCase).BackfillDaysPlayed(t.Context(), //nolint:wrapcheck // the tests read the error.
		connect.NewRequest(&progressionv1.BackfillDaysPlayedRequest{DryRun: dryRun}))
}

func TestTheAnswerSaysHowManyAccountsWereReadAndChanged(t *testing.T) {
	dryRuns := []bool{}
	useCase := stubUseCase{backfilled: backfill_days_played_usecase.Backfilled{Accounts: 1200, Changed: 37}, dryRuns: &dryRuns}

	res, err := backfill(t, useCase, true)

	require.NoError(t, err)
	assert.Equal(t, uint32(1200), res.Msg.GetAccounts())
	assert.Equal(t, uint32(37), res.Msg.GetChanged())

	_, err = backfill(t, useCase, false)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false}, dryRuns, "the dry run is the caller's")
}

func TestAFailureIsAnError(t *testing.T) {
	dryRuns := []bool{}
	useCase := stubUseCase{err: errors.New("planet is down"), dryRuns: &dryRuns}

	_, err := backfill(t, useCase, false)

	assert.ErrorIs(t, err, useCase.err)
}
