package backfill_titles_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/backfill_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/backfill_titles_usecase"
)

type stubUseCase struct {
	backfill backfill_titles_usecase.Backfill
	err      error
}

func (s stubUseCase) Execute(context.Context) (backfill_titles_usecase.Backfill, error) {
	return s.backfill, s.err
}

func backfill(t *testing.T, useCase stubUseCase) (*connect.Response[playerv1.BackfillTitlesResponse], error) {
	t.Helper()

	return backfill_titles_handler.New(useCase).BackfillTitles(t.Context(), //nolint:wrapcheck // the tests read the error.
		connect.NewRequest(&playerv1.BackfillTitlesRequest{}))
}

func TestTheAnswerSaysHowManyAccountsEarnATitle(t *testing.T) {
	res, err := backfill(t, stubUseCase{backfill: backfill_titles_usecase.Backfill{Accounts: 42}})

	require.NoError(t, err)
	assert.Equal(t, uint32(42), res.Msg.GetAccounts())
}

func TestAFailureIsAnError(t *testing.T) {
	_, err := backfill(t, stubUseCase{err: errors.New("postgres is down")})

	assert.Error(t, err)
}
