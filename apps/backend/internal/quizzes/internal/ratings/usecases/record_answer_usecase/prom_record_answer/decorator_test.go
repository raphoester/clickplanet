package prom_record_answer_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings"
	"github.com/raphoester/clickplanet.lol-backend/internal/quizzes/internal/ratings/usecases/record_answer_usecase/prom_record_answer"
)

type stubUseCase struct {
	err error
}

func (s *stubUseCase) Execute(context.Context, ratings.Answer) error { return s.err }

func TestAnAnswerNotCountedIsCountedApart(t *testing.T) {
	registry := prometheus.NewRegistry()
	useCase := &stubUseCase{}
	decorator := prom_record_answer.New(useCase, registry)

	for _, err := range []error{nil, nil, errors.New("postgres is down")} {
		useCase.err = err
		_ = decorator.Execute(t.Context(), ratings.Answer{})
	}

	require.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(`
# HELP quiz_answers_recorded_total Quiz answers counted for their question's rating, by outcome: recorded, or failed
# TYPE quiz_answers_recorded_total counter
quiz_answers_recorded_total{outcome="failed"} 1
quiz_answers_recorded_total{outcome="recorded"} 2
`), "quiz_answers_recorded_total"))
}
