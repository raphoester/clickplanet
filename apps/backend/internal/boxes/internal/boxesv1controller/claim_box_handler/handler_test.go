package claim_box_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	boxesv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/boxes/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxes/usecases/claim_box_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/boxes/internal/boxesv1controller/claim_box_handler"
)

type stubUseCase struct {
	err error

	in claim_box_usecase.In
}

func (s *stubUseCase) Execute(_ context.Context, in claim_box_usecase.In) error {
	s.in = in
	return s.err
}

func claim(t *testing.T, useCase claim_box_handler.UseCase) (*boxesv1.ClaimBoxResponse, error) {
	t.Helper()

	res, err := claim_box_handler.New(useCase).ClaimBox(t.Context(),
		connect.NewRequest(&boxesv1.ClaimBoxRequest{Token: "a-token"}))
	if err != nil {
		return nil, fmt.Errorf("claim refused: %w", err)
	}

	return res.Msg, nil
}

func TestTheTokenReachesTheUseCaseAndTheAnswerSaysNothingOfTheBox(t *testing.T) {
	useCase := &stubUseCase{}

	msg, err := claim(t, useCase)
	require.NoError(t, err)

	assert.Equal(t, "a-token", useCase.in.Token)
	assert.NotNil(t, msg)
}

func TestARefusedClaimIsNotFoundAndSaysNothingAboutWhy(t *testing.T) {
	_, err := claim(t, &stubUseCase{err: fmt.Errorf("lapsed, or never yours: %w", claim_box_usecase.ErrNoSuchBox)})

	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.Contains(t, err.Error(), claim_box_usecase.ErrNoSuchBox.Error())
	assert.NotContains(t, err.Error(), "lapsed")
}

func TestAFailedTellIsLeftToTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	_, err := claim(t, &stubUseCase{err: down})

	require.ErrorIs(t, err, down)
	assert.NotEqual(t, connect.CodeNotFound, connect.CodeOf(err))
}
