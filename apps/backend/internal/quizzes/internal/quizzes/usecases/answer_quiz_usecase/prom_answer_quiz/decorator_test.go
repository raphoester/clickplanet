package prom_answer_quiz_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/answer_quiz_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/quizzes/usecases/answer_quiz_usecase/prom_answer_quiz"
)

type stubUseCase struct {
	out answer_quiz_usecase.Out
	err error
}

func (s *stubUseCase) Execute(context.Context, answer_quiz_usecase.In) (answer_quiz_usecase.Out, error) {
	return s.out, s.err
}

func TestAFailedAnswerIsCountedApartFromARefusal(t *testing.T) {
	registry := prometheus.NewRegistry()
	useCase := &stubUseCase{}
	decorator, _ := prom_answer_quiz.New(useCase, registry)

	for _, answered := range []stubUseCase{
		{out: answer_quiz_usecase.Out{Correct: true}},
		{},
		{err: answer_quiz_usecase.ErrNoSuchQuiz},
		{err: errors.New("postgres is down")},
	} {
		*useCase = answered
		_, _ = decorator.Execute(t.Context(), answer_quiz_usecase.In{})
	}

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP quiz_answers_total Quizzes answered, by outcome
# TYPE quiz_answers_total counter
quiz_answers_total{outcome="correct"} 1
quiz_answers_total{outcome="failed"} 1
quiz_answers_total{outcome="refused"} 1
quiz_answers_total{outcome="wrong"} 1
`), "quiz_answers_total"))
}
