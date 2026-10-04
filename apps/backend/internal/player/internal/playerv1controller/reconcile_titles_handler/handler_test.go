package reconcile_titles_handler_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/reconcile_titles_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles/usecases/reconcile_titles_usecase"
)

type stubUseCase struct {
	reconciled reconcile_titles_usecase.Reconciled
	err        error
}

func (s stubUseCase) Execute(context.Context) (reconcile_titles_usecase.Reconciled, error) {
	return s.reconciled, s.err
}

func reconcile(t *testing.T, useCase stubUseCase) (*connect.Response[playerv1.ReconcileTitlesResponse], error) {
	t.Helper()

	return reconcile_titles_handler.New(useCase).ReconcileTitles(t.Context(), //nolint:wrapcheck // the tests read the error.
		connect.NewRequest(&playerv1.ReconcileTitlesRequest{}))
}

func TestTheAnswerSaysHowManyTitlesWereGrantedAndRevoked(t *testing.T) {
	res, err := reconcile(t, stubUseCase{reconciled: reconcile_titles_usecase.Reconciled{Granted: 42, Revoked: 7}})

	require.NoError(t, err)
	assert.Equal(t, uint32(42), res.Msg.GetGranted())
	assert.Equal(t, uint32(7), res.Msg.GetRevoked())
}

func TestAFailureIsAnError(t *testing.T) {
	_, err := reconcile(t, stubUseCase{err: errors.New("postgres is down")})

	assert.Error(t, err)
}
