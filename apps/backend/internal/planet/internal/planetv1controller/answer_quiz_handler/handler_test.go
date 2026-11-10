package answer_quiz_handler_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/answer_quiz_handler"
)

type stubUseCase struct {
	err error
}

func (s stubUseCase) Execute(context.Context, answer_quiz_usecase.In) (answer_quiz_usecase.Out, error) {
	return answer_quiz_usecase.Out{}, s.err
}

func answer(t *testing.T, useCase answer_quiz_handler.UseCase) error {
	t.Helper()

	_, err := answer_quiz_handler.New(useCase).AnswerQuiz(t.Context(),
		connect.NewRequest(&planetv1.AnswerQuizRequest{Token: "a-token", Choice: 1, CountryId: "fr"}))
	if err != nil {
		return fmt.Errorf("answer refused: %w", err)
	}
	return nil
}

func TestARefusedAnswerIsNotFound(t *testing.T) {
	err := answer(t, stubUseCase{err: answer_quiz_usecase.ErrNoSuchQuiz})

	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAFailedGrantIsLeftToTheErrorNet(t *testing.T) {
	down := errors.New("postgres is down")

	err := answer(t, stubUseCase{err: down})

	require.ErrorIs(t, err, down)
	assert.NotEqual(t, connect.CodeNotFound, connect.CodeOf(err))
}
